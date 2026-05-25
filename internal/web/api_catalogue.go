package web

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/noble-ch/inventory-optimizer/internal/analytics"
	"github.com/noble-ch/inventory-optimizer/internal/engine"
	"github.com/noble-ch/inventory-optimizer/internal/models"
	"github.com/noble-ch/inventory-optimizer/internal/store"
)

type skuHistoryResponse struct {
	SKU       *store.SKU                `json:"sku"`
	Sales     []store.SalesEntry        `json:"sales"`
	Movements []store.InventoryMovement `json:"movements"`
}

// ---------------------------------------------------------------------------
// REST API Handlers — Catalogue (SKUs & Sales)
// ---------------------------------------------------------------------------

func (s *Server) requirePremium(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := s.currentUser(r)
		if claims == nil {
			s.sendErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		if s.db == nil {
			s.sendErrorJSON(w, http.StatusServiceUnavailable, "Database not configured")
			return
		}

		state := s.accessStateForUser(r)
		if state.AccountStatus != "premium" && state.AccountStatus != "trial" {
			message := "Your free 6-month trial has ended. Subscribe to continue using premium features."
			if state.PremiumExpired {
				message = "Your free trial has expired. Subscribe to continue using premium features."
			}
			s.sendErrorJSON(w, http.StatusPaymentRequired, message)
			return
		}

		next(w, r)
	}
}

func (s *Server) handleAPIGetSKUs(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	skus, err := s.db.GetSKUs(r.Context(), claims.Subject)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to load SKUs")
		return
	}

	// Optional server-side filters via query params
	minStock := -1
	maxStock := -1
	if v := r.URL.Query().Get("min_stock"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			minStock = n
		}
	}
	if v := r.URL.Query().Get("max_stock"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxStock = n
		}
	}
	abcFilter := strings.ToUpper(r.URL.Query().Get("abc"))
	xyzFilter := strings.ToUpper(r.URL.Query().Get("xyz"))
	sortBy := r.URL.Query().Get("sort")

	// If ABC/XYZ filters requested, compute classifications
	var classifications map[string]analytics.SKUClassification
	if abcFilter != "" || xyzFilter != "" {
		sales, _ := s.db.GetSalesEntries(r.Context(), claims.Subject)
		classifications = analytics.ClassifyCatalogue(skus, sales)
	}

	// Apply filters
	var out []store.SKU
	for _, sku := range skus {
		if minStock >= 0 && sku.CurrentStock < minStock {
			continue
		}
		if maxStock >= 0 && sku.CurrentStock > maxStock {
			continue
		}
		if classifications != nil {
			if c, ok := classifications[sku.SKUID]; ok {
				if abcFilter != "" && c.ABCClass != abcFilter {
					continue
				}
				if xyzFilter != "" && c.XYZClass != xyzFilter {
					continue
				}
			} else {
				// no classification for this sku — skip if filtering requested
				if abcFilter != "" || xyzFilter != "" {
					continue
				}
			}
		}
		out = append(out, sku)
	}

	// Simple sorts
	switch sortBy {
	case "sku_asc":
		// keep as-is; DB already returns created_at order
	case "sku_desc":
		// reverse
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	case "stock_asc":
		// simple bubble (small lists expected)
		for i := 0; i < len(out); i++ {
			for j := i + 1; j < len(out); j++ {
				if out[i].CurrentStock > out[j].CurrentStock {
					out[i], out[j] = out[j], out[i]
				}
			}
		}
	case "stock_desc":
		for i := 0; i < len(out); i++ {
			for j := i + 1; j < len(out); j++ {
				if out[i].CurrentStock < out[j].CurrentStock {
					out[i], out[j] = out[j], out[i]
				}
			}
		}
	}

	if out == nil {
		out = []store.SKU{}
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{"skus": out})
}

func (s *Server) handleAPICreateSKU(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)

	var sku store.SKU
	if err := json.NewDecoder(r.Body).Decode(&sku); err != nil {
		s.sendErrorJSON(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if sku.SKUID == "" {
		s.sendErrorJSON(w, http.StatusBadRequest, "SKU ID is required")
		return
	}

	sku.UserID = claims.Subject

	if err := s.db.CreateSKU(r.Context(), &sku); err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to save SKU")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]string{"message": "SKU saved"})
}

func (s *Server) handleAPIGetSKUDetail(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	skuID := r.PathValue("id")
	if skuID == "" {
		s.sendErrorJSON(w, http.StatusBadRequest, "Missing SKU ID")
		return
	}

	sku, err := s.db.GetSKU(r.Context(), claims.Subject, skuID)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to load SKU")
		return
	}
	if sku == nil {
		s.sendErrorJSON(w, http.StatusNotFound, "SKU not found")
		return
	}

	sales, err := s.db.GetSalesEntries(r.Context(), claims.Subject)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to load sales history")
		return
	}
	filteredSales := make([]store.SalesEntry, 0)
	for _, sale := range sales {
		if sale.SKUID == skuID {
			filteredSales = append(filteredSales, sale)
		}
	}

	movements, err := s.db.GetInventoryMovements(r.Context(), claims.Subject, skuID)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to load movement history")
		return
	}

	s.sendJSON(w, http.StatusOK, skuHistoryResponse{SKU: sku, Sales: filteredSales, Movements: movements})
}

func (s *Server) handleAPIDeleteSKU(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	skuID := r.PathValue("id")
	if skuID == "" {
		s.sendErrorJSON(w, http.StatusBadRequest, "Missing SKU ID")
		return
	}

	if err := s.db.DeleteSKU(r.Context(), claims.Subject, skuID); err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to delete SKU")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]string{"message": "SKU deleted"})
}

func (s *Server) handleAPILogSales(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	skuID := r.PathValue("id")

	var payload struct {
		Date     string `json:"date"`
		Quantity int    `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.sendErrorJSON(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	d, err := time.Parse("2006-01-02", payload.Date)
	if err != nil {
		// fallback mapping default today if invalid
		d = time.Now()
	}

	sku, err := s.db.RecordSale(r.Context(), claims.Subject, skuID, payload.Quantity, d)
	if err != nil {
		s.sendErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{"message": "Sales logged", "sku": sku})
}

func (s *Server) handleAPIReplenishSKU(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	skuID := r.PathValue("id")
	if skuID == "" {
		s.sendErrorJSON(w, http.StatusBadRequest, "Missing SKU ID")
		return
	}

	var payload struct {
		Date     string `json:"date"`
		Quantity int    `json:"quantity"`
		Note     string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.sendErrorJSON(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	if payload.Quantity <= 0 {
		s.sendErrorJSON(w, http.StatusBadRequest, "Quantity must be greater than zero")
		return
	}

	movementDate := time.Now()
	if payload.Date != "" {
		if parsed, err := time.Parse("2006-01-02", payload.Date); err == nil {
			movementDate = parsed
		}
	}

	sku, err := s.db.RecordReplenishment(r.Context(), claims.Subject, skuID, payload.Quantity, movementDate, payload.Note)
	if err != nil {
		s.sendErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{"message": "Stock replenished", "sku": sku})
}

func (s *Server) handleAPIExportCatalogueCSV(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	skus, err := s.db.GetSKUs(r.Context(), claims.Subject)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to load SKUs")
		return
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Write([]string{"sku_id", "name", "unit_cost", "order_cost", "holding_pct", "lead_time_days", "selling_price", "current_stock", "created_at"})
	for _, sku := range skus {
		writer.Write([]string{
			sku.SKUID,
			sku.Name,
			strconv.FormatFloat(sku.UnitCost, 'f', -1, 64),
			strconv.FormatFloat(sku.OrderCost, 'f', -1, 64),
			strconv.FormatFloat(sku.HoldingPct, 'f', -1, 64),
			strconv.Itoa(sku.LeadTimeDays),
			strconv.FormatFloat(sku.SellingPrice, 'f', -1, 64),
			strconv.Itoa(sku.CurrentStock),
			sku.CreatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=catalogue.csv")
	w.Write(buf.Bytes())
}

func (s *Server) handleAPIAutoAnalyze(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)

	skus, err := s.db.GetSKUs(r.Context(), claims.Subject)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to get SKUs")
		return
	}
	if len(skus) == 0 {
		s.sendErrorJSON(w, http.StatusBadRequest, "No SKUs in catalogue")
		return
	}

	sales, err := s.db.GetSalesEntries(r.Context(), claims.Subject)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to get sales entries")
		return
	}

	var salesBuf bytes.Buffer
	salesWriter := csv.NewWriter(&salesBuf)
	salesWriter.Write([]string{"sku", "week", "units_sold"})
	for _, entry := range sales {
		salesWriter.Write([]string{
			entry.SKUID,
			entry.Date.Format("2006-01-02"),
			strconv.Itoa(entry.Quantity),
		})
	}
	salesWriter.Flush()

	var paramsBuf bytes.Buffer
	paramsWriter := csv.NewWriter(&paramsBuf)
	paramsWriter.Write([]string{"sku", "current_inventory", "lead_time_days", "unit_cost", "order_cost", "holding_cost_rate"})
	for _, sku := range skus {
		holdingRate := sku.HoldingPct
		if holdingRate > 1 {
			holdingRate = holdingRate / 100
		}
		if holdingRate < 0 {
			holdingRate = 0
		}
		paramsWriter.Write([]string{
			sku.SKUID,
			strconv.Itoa(sku.CurrentStock),
			strconv.Itoa(sku.LeadTimeDays),
			strconv.FormatFloat(sku.UnitCost, 'f', -1, 64),
			strconv.FormatFloat(sku.OrderCost, 'f', -1, 64),
			strconv.FormatFloat(holdingRate, 'f', -1, 64),
		})
	}
	paramsWriter.Flush()
	if err := paramsWriter.Error(); err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to prepare analysis parameters")
		return
	}
	if err := salesWriter.Error(); err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to prepare sales history")
		return
	}

	opts := engine.DefaultOptions()
	start := time.Now()
	reports, warnings, err := engine.RunFromReaders(&salesBuf, &paramsBuf, opts)
	elapsed := time.Since(start)

	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Analysis failed: "+err.Error())
		return
	}

	// Add functionality to immediately save report if desired
	title := "Catalogue Auto-Analysis"
	dbReport := &store.Report{
		UserID:       claims.Subject,
		Title:        title,
		ServiceLevel: opts.ServiceLevel,
		SimRuns:      opts.SimRuns,
		SimWeeks:     opts.SimWeeks,
		SKUCount:     len(reports),
		Warnings:     warnings,
		Results:      reports,
	}

	response := map[string]interface{}{
		"skus_analyzed": len(reports),
		"warnings":      warnings,
		"elapsed_ms":    elapsed.Milliseconds(),
		"results":       reports,
	}
	if response["warnings"] == nil {
		response["warnings"] = []string{}
	}
	if reports == nil {
		reports = []models.SKUReport{}
		response["results"] = reports
	}

	if err := s.db.CreateReport(r.Context(), dbReport); err == nil {
		response["saved_report_id"] = dbReport.ID
	}

	s.sendJSON(w, http.StatusOK, response)
}
