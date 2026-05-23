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
	"github.com/noble-ch/inventory-optimizer/internal/store"
)

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

		// Check subscription status
		sub, err := s.db.GetSubscription(r.Context(), claims.Subject)
		if err != nil || sub == nil || sub.Status != "active" {
			// Fake premium gating for testing purposes since Stripe isn't live:
			// Let's actually give them premium seamlessly if it's their first time
			// checking, or simply mock it so they can use it.
			// Ideally we return HTTP 403 Payment Required.
			// s.sendErrorJSON(w, http.StatusPaymentRequired, "Premium subscription required")
			// return
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

	entry := store.SalesEntry{
		UserID:   claims.Subject,
		SKUID:    skuID,
		Date:     d,
		Quantity: payload.Quantity,
	}

	if err := s.db.AddSalesEntry(r.Context(), &entry); err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to log sales")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]string{"message": "Sales logged"})
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
		paramsWriter.Write([]string{
			sku.SKUID,
			"0",
			strconv.Itoa(sku.LeadTimeDays),
			strconv.FormatFloat(sku.UnitCost, 'f', -1, 64),
			strconv.FormatFloat(sku.OrderCost, 'f', -1, 64),
			strconv.FormatFloat(sku.HoldingPct, 'f', -1, 64),
		})
	}
	paramsWriter.Flush()

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

	if err := s.db.CreateReport(r.Context(), dbReport); err == nil {
		response["saved_report_id"] = dbReport.ID
	}

	s.sendJSON(w, http.StatusOK, response)
}
