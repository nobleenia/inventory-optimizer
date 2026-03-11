package api

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/noble-ch/inventory-optimizer/internal/auth"
	"github.com/noble-ch/inventory-optimizer/internal/engine"
	"github.com/noble-ch/inventory-optimizer/internal/models"
	"github.com/noble-ch/inventory-optimizer/internal/reporting"
	"github.com/noble-ch/inventory-optimizer/internal/store"
)

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": models.Version,
	})
}

// ---------------------------------------------------------------------------
// Auth: register / login / refresh
// ---------------------------------------------------------------------------

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User   store.User     `json:"user"`
	Tokens auth.TokenPair `json:"tokens"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "A valid email address is required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	user, err := s.db.CreateUser(r.Context(), req.Email, hash)
	if errors.Is(err, store.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "Email already registered")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create account")
		return
	}

	tokens, err := s.auth.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{User: *user, Tokens: *tokens})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	user, err := s.db.GetUserByEmail(r.Context(), req.Email)
	if errors.Is(err, store.ErrUserNotFound) {
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Login failed")
		return
	}

	if err := auth.CheckPassword(user.Password, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	tokens, err := s.auth.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{User: *user, Tokens: *tokens})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	claims, err := s.auth.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	// Verify user still exists.
	user, err := s.db.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "User not found")
		return
	}

	tokens, err := s.auth.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	writeJSON(w, http.StatusOK, auth.TokenPair{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// ---------------------------------------------------------------------------
// Profile
// ---------------------------------------------------------------------------

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	uid, _ := userFromContext(r.Context())
	user, err := s.db.GetUserByID(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// ---------------------------------------------------------------------------
// Analyze — upload CSVs, run engine, persist report
// ---------------------------------------------------------------------------

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	uid, _ := userFromContext(r.Context())

	// 10 MB limit.
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Upload too large (max 10 MB)")
		return
	}

	salesFile, _, err := r.FormFile("sales_file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Missing sales_file in multipart form")
		return
	}
	defer salesFile.Close()

	paramsFile, _, err := r.FormFile("params_file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Missing params_file in multipart form")
		return
	}
	defer paramsFile.Close()

	// Optional fields.
	title := r.FormValue("title")
	if title == "" {
		title = "Untitled Analysis"
	}

	sl := 0.95
	if v := r.FormValue("service_level"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			sl = parsed
		}
	}
	simRuns := 500
	if v := r.FormValue("sim_runs"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			simRuns = parsed
		}
	}
	simWeeks := 52
	if v := r.FormValue("sim_weeks"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			simWeeks = parsed
		}
	}

	opts := engine.Options{
		ServiceLevel: sl,
		SimRuns:      simRuns,
		SimWeeks:     simWeeks,
	}

	reports, warnings, err := engine.RunFromReaders(salesFile, paramsFile, opts)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	// Persist report.
	report := &store.Report{
		UserID:       uid,
		Title:        title,
		ServiceLevel: sl,
		SimRuns:      simRuns,
		SimWeeks:     simWeeks,
		SKUCount:     len(reports),
		Warnings:     warnings,
		Results:      reports,
	}

	if err := s.db.CreateReport(r.Context(), report); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save report")
		return
	}

	writeJSON(w, http.StatusCreated, report)
}

// ---------------------------------------------------------------------------
// Reports — list / get / delete / CSV download
// ---------------------------------------------------------------------------

func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	uid, _ := userFromContext(r.Context())

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	reports, total, err := s.db.ListReports(r.Context(), uid, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list reports")
		return
	}
	if reports == nil {
		reports = []store.Report{}
	}

	writeJSONWithMeta(w, http.StatusOK, reports, &meta{
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	uid, _ := userFromContext(r.Context())
	reportID := r.PathValue("id")

	report, err := s.db.GetReport(r.Context(), uid, reportID)
	if errors.Is(err, store.ErrReportNotFound) {
		writeError(w, http.StatusNotFound, "Report not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve report")
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleDeleteReport(w http.ResponseWriter, r *http.Request) {
	uid, _ := userFromContext(r.Context())
	reportID := r.PathValue("id")

	if err := s.db.DeleteReport(r.Context(), uid, reportID); errors.Is(err, store.ErrReportNotFound) {
		writeError(w, http.StatusNotFound, "Report not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete report")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Report deleted"})
}

func (s *Server) handleReportCSV(w http.ResponseWriter, r *http.Request) {
	uid, _ := userFromContext(r.Context())
	reportID := r.PathValue("id")

	report, err := s.db.GetReport(r.Context(), uid, reportID)
	if errors.Is(err, store.ErrReportNotFound) {
		writeError(w, http.StatusNotFound, "Report not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve report")
		return
	}

	var buf bytes.Buffer
	if err := reporting.WriteCSV(&buf, report.Results); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate CSV")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report-%s.csv", reportID))
	w.Write(buf.Bytes())
}

func (s *Server) handleReportPDF(w http.ResponseWriter, r *http.Request) {
	uid, _ := userFromContext(r.Context())
	reportID := r.PathValue("id")

	report, err := s.db.GetReport(r.Context(), uid, reportID)
	if errors.Is(err, store.ErrReportNotFound) {
		writeError(w, http.StatusNotFound, "Report not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve report")
		return
	}

	var buf bytes.Buffer
	if err := reporting.WritePDF(&buf, report.Results); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate PDF")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report-%s.pdf", reportID))
	w.Write(buf.Bytes())
}
