// Package api implements the JSON REST API for the inventory optimizer.
//
// It exposes endpoints for authentication, running analyses, and
// managing persisted reports. All handlers consume the store and auth
// packages — they never touch CSV parsing or simulation directly,
// delegating that to the engine package.
package api

import (
	"net/http"
	"time"

	"github.com/noble-ch/inventory-optimizer/internal/auth"
	"github.com/noble-ch/inventory-optimizer/internal/store"
)

// Server holds dependencies and exposes the HTTP handler.
type Server struct {
	db      *store.DB
	auth    *auth.Service
	limiter *RateLimiter
	mux     *http.ServeMux
}

// NewServer creates a configured API server.
// Rate limit: 60 requests per minute with a burst of 60.
func NewServer(db *store.DB, authSvc *auth.Service) *Server {
	s := &Server{
		db:      db,
		auth:    authSvc,
		limiter: NewRateLimiter(60, 60, 1*time.Minute),
		mux:     http.NewServeMux(),
	}
	s.routes()
	return s
}

// Handler returns the top-level http.Handler with global middleware applied.
func (s *Server) Handler() http.Handler {
	return chainMiddleware(s.mux,
		corsMiddleware,
		logMiddleware,
		s.limiter.Middleware,
	)
}

// routes registers all API endpoints.
func (s *Server) routes() {
	// Public — no auth required.
	s.mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/auth/refresh", s.handleRefresh)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)

	// Protected — JWT required.
	s.mux.Handle("POST /api/analyze", s.requireAuth(http.HandlerFunc(s.handleAnalyze)))
	s.mux.Handle("GET /api/reports", s.requireAuth(http.HandlerFunc(s.handleListReports)))
	s.mux.Handle("GET /api/reports/{id}", s.requireAuth(http.HandlerFunc(s.handleGetReport)))
	s.mux.Handle("DELETE /api/reports/{id}", s.requireAuth(http.HandlerFunc(s.handleDeleteReport)))
	s.mux.Handle("GET /api/reports/{id}/csv", s.requireAuth(http.HandlerFunc(s.handleReportCSV)))
	s.mux.Handle("GET /api/reports/{id}/pdf", s.requireAuth(http.HandlerFunc(s.handleReportPDF)))
	s.mux.Handle("GET /api/user/profile", s.requireAuth(http.HandlerFunc(s.handleProfile)))
}
