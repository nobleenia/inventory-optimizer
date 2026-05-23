package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/noble-ch/inventory-optimizer/internal/auth"
	"github.com/noble-ch/inventory-optimizer/internal/engine"
	"github.com/noble-ch/inventory-optimizer/internal/reporting"
	"github.com/noble-ch/inventory-optimizer/internal/store"
)

func (s *Server) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) sendErrorJSON(w http.ResponseWriter, status int, message string) {
	s.sendJSON(w, status, map[string]string{"error": message})
}

// ---------------------------------------------------------------------------
// REST API Handlers — Auth
// ---------------------------------------------------------------------------

func (s *Server) handleAPIRegister(w http.ResponseWriter, r *http.Request) {
	if s.db == nil || s.auth == nil {
		s.sendErrorJSON(w, http.StatusServiceUnavailable, "Auth is not configured")
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendErrorJSON(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if req.Email == "" || len(req.Password) < 8 {
		s.sendErrorJSON(w, http.StatusBadRequest, "Email required and password must be >= 8 chars")
		return
	}

	user, err := s.db.CreateUser(r.Context(), req.Email, req.Password)
	if err != nil {
		s.sendErrorJSON(w, http.StatusConflict, "Email already exists or registration failed")
		return
	}

	tokens, err := s.auth.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	s.sendJSON(w, http.StatusCreated, tokens)
}

func (s *Server) handleAPILogin(w http.ResponseWriter, r *http.Request) {
	if s.db == nil || s.auth == nil {
		s.sendErrorJSON(w, http.StatusServiceUnavailable, "Auth is not configured")
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendErrorJSON(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	user, err := s.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		s.sendErrorJSON(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if err := auth.CheckPassword(user.Password, req.Password); err != nil {
		s.sendErrorJSON(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	tokens, err := s.auth.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	s.sendJSON(w, http.StatusOK, tokens)
}

// ---------------------------------------------------------------------------
// REST API Handlers — Analyze
// ---------------------------------------------------------------------------

func (s *Server) handleAPIAnalyze(w http.ResponseWriter, r *http.Request) {
	// Limit upload size to 10 MB.
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		s.sendErrorJSON(w, http.StatusBadRequest, "Upload too large or invalid multipart form")
		return
	}

	salesFile, _, err := r.FormFile("sales_file")
	if err != nil {
		s.sendErrorJSON(w, http.StatusBadRequest, "Missing sales_file")
		return
	}
	defer salesFile.Close()

	paramsFile, _, err := r.FormFile("params_file")
	if err != nil {
		s.sendErrorJSON(w, http.StatusBadRequest, "Missing params_file")
		return
	}
	defer paramsFile.Close()

	opts := engine.DefaultOptions()
	start := time.Now()
	reports, warnings, err := engine.RunFromReaders(salesFile, paramsFile, opts)
	elapsed := time.Since(start)

	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"skus_analyzed": len(reports),
		"warnings":      warnings,
		"elapsed_ms":    elapsed.Milliseconds(),
		"results":       reports,
	}

	// If authenticated, optionally save it
	claims := s.currentUser(r)
	if claims != nil && s.db != nil {
		title := r.FormValue("title")
		if title == "" {
			title = "API Analysis - " + time.Now().Format("2006-01-02 15:04")
		}

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

		if err := s.db.CreateReport(r.Context(), dbReport); err == nil {
			response["saved_report_id"] = dbReport.ID
		}
	} else if claims == nil {
		// Truncate for guests in API too to mirror web.
		if len(reports) > maxGuestSKUs {
			response["results"] = reports[:maxGuestSKUs]
			response["guest_locked_skus"] = len(reports) - maxGuestSKUs
		}
	}

	s.sendJSON(w, http.StatusOK, response)
}

// ---------------------------------------------------------------------------
// REST API Handlers — Saved Reports
// ---------------------------------------------------------------------------

func (s *Server) handleAPIReportsList(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	if claims == nil {
		s.sendErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if s.db == nil {
		s.sendErrorJSON(w, http.StatusServiceUnavailable, "Database not configured")
		return
	}

	q := r.URL.Query().Get("q")
	sortBy := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil {
			offset = n
		}
	}

	reports, total, err := s.db.ListReports(r.Context(), claims.Subject, limit, offset, q, sortBy, order)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to fetch reports")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"reports": reports,
		"total":   total,
	})
}

func (s *Server) handleAPIReportDetail(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	if claims == nil {
		s.sendErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if s.db == nil {
		s.sendErrorJSON(w, http.StatusServiceUnavailable, "Database not configured")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		s.sendErrorJSON(w, http.StatusBadRequest, "Missing report ID")
		return
	}

	rep, err := s.db.GetReport(r.Context(), claims.Subject, id)
	if err != nil {
		s.sendErrorJSON(w, http.StatusNotFound, "Report not found")
		return
	}

	s.sendJSON(w, http.StatusOK, rep)
}

// handleAPIReportCSV streams a CSV for a saved report.
func (s *Server) handleAPIReportCSV(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	if claims == nil {
		s.sendErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if s.db == nil {
		s.sendErrorJSON(w, http.StatusServiceUnavailable, "Database not configured")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		s.sendErrorJSON(w, http.StatusBadRequest, "Missing report ID")
		return
	}
	rep, err := s.db.GetReport(r.Context(), claims.Subject, id)
	if err != nil {
		s.sendErrorJSON(w, http.StatusNotFound, "Report not found")
		return
	}
	var buf bytes.Buffer
	if err := reporting.WriteCSV(&buf, rep.Results); err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to generate CSV")
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report-%s.csv", id))
	w.Write(buf.Bytes())
}

// handleAPIReportPDF streams a PDF for a saved report.
func (s *Server) handleAPIReportPDF(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	if claims == nil {
		s.sendErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if s.db == nil {
		s.sendErrorJSON(w, http.StatusServiceUnavailable, "Database not configured")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		s.sendErrorJSON(w, http.StatusBadRequest, "Missing report ID")
		return
	}
	rep, err := s.db.GetReport(r.Context(), claims.Subject, id)
	if err != nil {
		s.sendErrorJSON(w, http.StatusNotFound, "Report not found")
		return
	}
	var buf bytes.Buffer
	if err := reporting.WritePDF(&buf, rep.Results); err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to generate PDF")
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report-%s.pdf", id))
	w.Write(buf.Bytes())
}

func (s *Server) handleAPIReportDelete(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	if claims == nil {
		s.sendErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if s.db == nil {
		s.sendErrorJSON(w, http.StatusServiceUnavailable, "Database not configured")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		s.sendErrorJSON(w, http.StatusBadRequest, "Missing report ID")
		return
	}

	if err := s.db.DeleteReport(r.Context(), claims.Subject, id); err != nil {
		s.sendErrorJSON(w, http.StatusNotFound, "Report not found or already deleted")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}
