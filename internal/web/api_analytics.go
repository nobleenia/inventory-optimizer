package web

import (
	"encoding/json"
	"net/http"

	"github.com/noble-ch/inventory-optimizer/internal/analytics"
	"github.com/noble-ch/inventory-optimizer/internal/models"
)

// handleAPIGetABCXYZ computes and returns the ABC/XYZ matrix
func (s *Server) handleAPIGetABCXYZ(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)

	skus, err := s.db.GetSKUs(r.Context(), claims.Subject)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to get SKUs")
		return
	}

	sales, err := s.db.GetSalesEntries(r.Context(), claims.Subject)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to get sales")
		return
	}

	classifications := analytics.ClassifyCatalogue(skus, sales)
	s.sendJSON(w, http.StatusOK, classifications)
}

// handleAPIGetForecast computes SES forecast for a specific SKU
func (s *Server) handleAPIGetForecast(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	skuID := r.PathValue("id")

	sales, err := s.db.GetSalesEntries(r.Context(), claims.Subject)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to get sales")
		return
	}

	// Filter for SKU and aggregate weekly
	weeklyDemands := make(map[string]int)
	for _, s := range sales {
		if s.SKUID == skuID {
			year, week := s.Date.ISOWeek()
			key := string(rune(year)) + "-" + string(rune(week))
			weeklyDemands[key] += s.Quantity
		}
	}

	var demands []float64
	for _, v := range weeklyDemands {
		demands = append(demands, float64(v))
	}

	// If zero demand history, return empty array
	if len(demands) == 0 {
		s.sendJSON(w, http.StatusOK, []analytics.ForecastPoint{})
		return
	}

	forecasts := analytics.ForecastSES(demands, 12, 0.3) // 12 weeks, alpha=0.3
	s.sendJSON(w, http.StatusOK, forecasts)
}

// handleAPIBudgetOptimize allocates a budget across the catalogue
func (s *Server) handleAPIBudgetOptimize(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)

	var payload struct {
		Budget float64 `json:"budget"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Budget <= 0 {
		s.sendErrorJSON(w, http.StatusBadRequest, "Invalid budget payload")
		return
	}

	skus, err := s.db.GetSKUs(r.Context(), claims.Subject)
	if err != nil || len(skus) == 0 {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to calculate: No SKUs")
		return
	}

	reports, _, err := s.db.ListReports(r.Context(), claims.Subject, 1, 0, "", "", "")
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to load recent analysis")
		return
	}
	if len(reports) == 0 {
		s.sendErrorJSON(w, http.StatusBadRequest, "Please run Auto-Analyze first")
		return
	}

	reportDetail, err := s.db.GetReport(r.Context(), claims.Subject, reports[0].ID)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to get latest report data")
		return
	}

	resultMap := make(map[string]models.SKUReport, len(reportDetail.Results))
	for _, result := range reportDetail.Results {
		resultMap[result.Parameters.SKU] = result
	}

	optimized := analytics.OptimizeBudget(payload.Budget, skus, resultMap)
	s.sendJSON(w, http.StatusOK, optimized)
}
