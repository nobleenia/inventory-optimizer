// Package web provides an HTTP server that wraps the inventory
// optimization engine behind a browser-friendly interface.
//
// Features:
//   - Guest mode: upload CSVs, view truncated results (1 SKU full, rest summary)
//   - Authenticated mode: full results, PDF/CSV downloads, save & manage reports
//   - Session management via JWT cookie (requires PostgreSQL + auth service)
package web

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/noble-ch/inventory-optimizer/internal/auth"
	"github.com/noble-ch/inventory-optimizer/internal/engine"
	"github.com/noble-ch/inventory-optimizer/internal/models"
	"github.com/noble-ch/inventory-optimizer/internal/reporting"
	"github.com/noble-ch/inventory-optimizer/internal/store"
)

//go:embed templates/*.html static/css/*.css static/js/*.js
var content embed.FS

const sessionCookieName = "io_session"
const maxGuestSKUs = 1 // guests see full detail for this many SKUs

// templateFuncs provides custom functions available in HTML templates.
var templateFuncs = template.FuncMap{
	"ceil":     func(f float64) float64 { return math.Ceil(f) },
	"pct":      func(f float64) string { return fmt.Sprintf("%.1f%%", f*100) },
	"currency": func(f float64) string { return fmt.Sprintf("€%.2f", f) },
	"fixed1":   func(f float64) string { return fmt.Sprintf("%.1f", f) },
	"fixed0":   func(f float64) string { return fmt.Sprintf("%.0f", f) },
	"upper":    strings.ToUpper,
	"mult":     func(a, b float64) float64 { return a * b },
	"json": func(v interface{}) template.JS {
		b, _ := json.Marshal(v)
		return template.JS(b)
	},
	"seq": func(n int) []int {
		s := make([]int, n)
		for i := range s {
			s[i] = i + 1
		}
		return s
	},
}

// Server holds the HTTP server configuration and optional auth/db deps.
type Server struct {
	Addr string
	mux  *http.ServeMux
	tmpl *template.Template
	db   *store.DB     // nil = no database (guest-only mode)
	auth *auth.Service // nil = no auth
}

// NewServer creates a configured server ready to ListenAndServe.
// Pass nil for db/authSvc to run in guest-only mode (no auth, no saved reports).
func NewServer(addr string, db *store.DB, authSvc *auth.Service) *Server {
	s := &Server{
		Addr: addr,
		mux:  http.NewServeMux(),
		db:   db,
		auth: authSvc,
	}

	// Parse embedded templates.
	s.tmpl = template.Must(
		template.New("").Funcs(templateFuncs).ParseFS(content, "templates/*.html"),
	)

	// Public routes.
	s.mux.HandleFunc("/", s.handleLanding)
	s.mux.HandleFunc("/upload", s.handleUpload)
	s.mux.HandleFunc("POST /analyze", s.handleAnalyze)
	s.mux.HandleFunc("/download/csv", s.handleDownloadCSV)
	s.mux.HandleFunc("/download/pdf", s.handleDownloadPDF)
	s.mux.Handle("/static/", http.FileServer(http.FS(content)))

	// Auth routes (only meaningful when db != nil, but always registered).
	s.mux.HandleFunc("GET /login", s.handleLoginPage)
	s.mux.HandleFunc("POST /login", s.handleLoginSubmit)
	s.mux.HandleFunc("GET /register", s.handleRegisterPage)
	s.mux.HandleFunc("POST /register", s.handleRegisterSubmit)
	s.mux.HandleFunc("GET /logout", s.handleLogout)

	// Authenticated routes.
	s.mux.HandleFunc("GET /reports", s.handleReportsList)
	s.mux.HandleFunc("GET /reports/{id}", s.handleReportDetail)
	s.mux.HandleFunc("POST /reports/{id}/delete", s.handleReportDelete)

	return s
}

// Start begins listening. It blocks until the server shuts down.
func (s *Server) Start() error {
	mode := "guest-only"
	if s.db != nil {
		mode = "full (auth + database)"
	}
	log.Printf("Starting web server on %s [%s mode]\n", s.Addr, mode)
	log.Printf("Open http://localhost%s in your browser.\n", s.Addr)
	return http.ListenAndServe(s.Addr, s.mux)
}

// hasAuth returns true when the server is configured with database + auth.
func (s *Server) hasAuth() bool { return s.db != nil && s.auth != nil }

// ---------------------------------------------------------------------------
// Session management
// ---------------------------------------------------------------------------

func (s *Server) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7, // 7 days
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// currentUser returns the authenticated user's claims, or nil.
func (s *Server) currentUser(r *http.Request) *auth.Claims {
	if s.auth == nil {
		return nil
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	claims, err := s.auth.ValidateAccessToken(cookie.Value)
	if err != nil {
		return nil
	}
	return claims
}

// baseData returns template data common to every page.
func (s *Server) baseData(r *http.Request) map[string]interface{} {
	d := map[string]interface{}{
		"Version":    models.Version,
		"HasAuth":    s.hasAuth(),
		"IsLoggedIn": false,
	}
	if claims := s.currentUser(r); claims != nil {
		d["User"] = claims
		d["UserEmail"] = claims.Email
		d["IsLoggedIn"] = true
	}
	return d
}

// ---------------------------------------------------------------------------
// Handlers — Landing, Upload
// ---------------------------------------------------------------------------

func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.render(w, "landing.html", s.baseData(r))
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	s.render(w, "index.html", s.baseData(r))
}

// ---------------------------------------------------------------------------
// Handlers — Auth
// ---------------------------------------------------------------------------

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if !s.hasAuth() {
		http.Redirect(w, r, "/upload", http.StatusSeeOther)
		return
	}
	d := s.baseData(r)
	d["Redirect"] = r.URL.Query().Get("redirect")
	s.render(w, "login.html", d)
}

func (s *Server) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if !s.hasAuth() {
		http.Redirect(w, r, "/upload", http.StatusSeeOther)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	d := s.baseData(r)
	d["Email"] = email
	d["Redirect"] = r.FormValue("redirect")

	if email == "" || password == "" {
		d["Error"] = "Please enter your email and password."
		s.render(w, "login.html", d)
		return
	}

	user, err := s.db.GetUserByEmail(r.Context(), email)
	if errors.Is(err, store.ErrUserNotFound) {
		d["Error"] = "Invalid email or password."
		s.render(w, "login.html", d)
		return
	}
	if err != nil {
		d["Error"] = "Something went wrong. Please try again."
		s.render(w, "login.html", d)
		return
	}

	if err := auth.CheckPassword(user.Password, password); err != nil {
		d["Error"] = "Invalid email or password."
		s.render(w, "login.html", d)
		return
	}

	tokens, err := s.auth.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		d["Error"] = "Failed to create session. Please try again."
		s.render(w, "login.html", d)
		return
	}

	s.setSessionCookie(w, tokens.AccessToken)

	redirect := r.FormValue("redirect")
	if redirect == "" || !strings.HasPrefix(redirect, "/") {
		redirect = "/upload"
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	if !s.hasAuth() {
		http.Redirect(w, r, "/upload", http.StatusSeeOther)
		return
	}
	s.render(w, "register.html", s.baseData(r))
}

func (s *Server) handleRegisterSubmit(w http.ResponseWriter, r *http.Request) {
	if !s.hasAuth() {
		http.Redirect(w, r, "/upload", http.StatusSeeOther)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	d := s.baseData(r)
	d["Email"] = email

	// Validation.
	if email == "" || !strings.Contains(email, "@") {
		d["Error"] = "Please enter a valid email address."
		s.render(w, "register.html", d)
		return
	}
	if len(password) < 8 {
		d["Error"] = "Password must be at least 8 characters."
		s.render(w, "register.html", d)
		return
	}
	if password != confirm {
		d["Error"] = "Passwords do not match."
		s.render(w, "register.html", d)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		d["Error"] = "Something went wrong. Please try again."
		s.render(w, "register.html", d)
		return
	}

	user, err := s.db.CreateUser(r.Context(), email, hash)
	if errors.Is(err, store.ErrEmailTaken) {
		d["Error"] = "An account with this email already exists."
		s.render(w, "register.html", d)
		return
	}
	if err != nil {
		d["Error"] = "Failed to create account. Please try again."
		s.render(w, "register.html", d)
		return
	}

	tokens, err := s.auth.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		d["Error"] = "Account created but failed to log in. Please use the login page."
		s.render(w, "register.html", d)
		return
	}

	s.setSessionCookie(w, tokens.AccessToken)
	http.Redirect(w, r, "/upload", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.clearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ---------------------------------------------------------------------------
// Handlers — Analyze
// ---------------------------------------------------------------------------

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	// Limit upload size to 10 MB.
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		s.renderError(w, r, "Upload too large. Maximum file size is 10 MB.")
		return
	}

	salesFile, _, err := r.FormFile("sales_file")
	if err != nil {
		s.renderError(w, r, "Please upload a sales history CSV file.")
		return
	}
	defer salesFile.Close()

	paramsFile, _, err := r.FormFile("params_file")
	if err != nil {
		s.renderError(w, r, "Please upload a SKU parameters CSV file.")
		return
	}
	defer paramsFile.Close()

	opts := engine.DefaultOptions()
	start := time.Now()
	reports, warnings, err := engine.RunFromReaders(salesFile, paramsFile, opts)
	elapsed := time.Since(start)

	if err != nil {
		s.renderError(w, r, fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	claims := s.currentUser(r)
	isGuest := claims == nil

	data := s.baseData(r)
	data["Reports"] = reports
	data["Warnings"] = warnings
	data["Elapsed"] = elapsed.Round(time.Millisecond).String()
	data["SKUCount"] = len(reports)
	data["IsGuest"] = isGuest
	data["MaxGuestSKUs"] = maxGuestSKUs

	// Authenticated users get full features.
	if !isGuest {
		// Generate temp CSV for download.
		tmpCSV := fmt.Sprintf("/tmp/inventory-report-%d.csv", time.Now().UnixNano())
		if err := reporting.ExportCSV(tmpCSV, reports); err != nil {
			log.Printf("CSV export warning: %v", err)
		} else {
			data["CSVPath"] = tmpCSV
		}

		// Generate temp PDF for download.
		tmpPDF := fmt.Sprintf("/tmp/inventory-report-%d.pdf", time.Now().UnixNano())
		if err := reporting.ExportPDF(tmpPDF, reports); err != nil {
			log.Printf("PDF export warning: %v", err)
		} else {
			data["PDFPath"] = tmpPDF
		}

		// Save report to database if available.
		if s.db != nil {
			title := r.FormValue("title")
			if title == "" {
				title = fmt.Sprintf("Analysis — %s", time.Now().Format("Jan 2, 2006 15:04"))
			}
			rpt := &store.Report{
				UserID:       claims.UserID,
				Title:        title,
				ServiceLevel: opts.ServiceLevel,
				SimRuns:      opts.SimRuns,
				SimWeeks:     opts.SimWeeks,
				SKUCount:     len(reports),
				Warnings:     warnings,
				Results:      reports,
			}
			if err := s.db.CreateReport(r.Context(), rpt); err != nil {
				log.Printf("Report save warning: %v", err)
			} else {
				data["SavedReportID"] = rpt.ID
			}
		}
	}

	s.render(w, "results.html", data)
}

// ---------------------------------------------------------------------------
// Handlers — Downloads
// ---------------------------------------------------------------------------

func (s *Server) handleDownloadCSV(w http.ResponseWriter, r *http.Request) {
	if s.currentUser(r) == nil {
		http.Redirect(w, r, "/login?redirect=/upload", http.StatusSeeOther)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" || !strings.HasPrefix(path, "/tmp/inventory-report-") {
		http.Error(w, "Invalid download path", http.StatusBadRequest)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "Report file not found. Please run the analysis again.", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=inventory-report.csv")
	w.Write(data)
}

func (s *Server) handleDownloadPDF(w http.ResponseWriter, r *http.Request) {
	if s.currentUser(r) == nil {
		http.Redirect(w, r, "/login?redirect=/upload", http.StatusSeeOther)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" || !strings.HasPrefix(path, "/tmp/inventory-report-") {
		http.Error(w, "Invalid download path", http.StatusBadRequest)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "Report file not found. Please run the analysis again.", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=inventory-report.pdf")
	w.Write(data)
}

// ---------------------------------------------------------------------------
// Handlers — Saved Reports
// ---------------------------------------------------------------------------

func (s *Server) handleReportsList(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	if claims == nil {
		http.Redirect(w, r, "/login?redirect=/reports", http.StatusSeeOther)
		return
	}
	if s.db == nil {
		http.Redirect(w, r, "/upload", http.StatusSeeOther)
		return
	}

	reports, total, err := s.db.ListReports(r.Context(), claims.UserID, 50, 0)
	if err != nil {
		s.renderError(w, r, "Failed to load reports.")
		return
	}

	d := s.baseData(r)
	d["SavedReports"] = reports
	d["TotalReports"] = total
	s.render(w, "reports.html", d)
}

func (s *Server) handleReportDetail(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	if claims == nil {
		http.Redirect(w, r, "/login?redirect="+r.URL.Path, http.StatusSeeOther)
		return
	}
	if s.db == nil {
		http.Redirect(w, r, "/upload", http.StatusSeeOther)
		return
	}

	reportID := r.PathValue("id")
	report, err := s.db.GetReport(r.Context(), claims.UserID, reportID)
	if errors.Is(err, store.ErrReportNotFound) {
		s.renderError(w, r, "Report not found.")
		return
	}
	if err != nil {
		s.renderError(w, r, "Failed to load report.")
		return
	}

	// Generate temp files for download.
	tmpCSV := fmt.Sprintf("/tmp/inventory-report-%d.csv", time.Now().UnixNano())
	if err := reporting.ExportCSV(tmpCSV, report.Results); err != nil {
		log.Printf("CSV export warning: %v", err)
	}
	tmpPDF := fmt.Sprintf("/tmp/inventory-report-%d.pdf", time.Now().UnixNano())
	if err := reporting.ExportPDF(tmpPDF, report.Results); err != nil {
		log.Printf("PDF export warning: %v", err)
	}

	d := s.baseData(r)
	d["Reports"] = report.Results
	d["Warnings"] = report.Warnings
	d["Elapsed"] = "saved report"
	d["SKUCount"] = report.SKUCount
	d["CSVPath"] = tmpCSV
	d["PDFPath"] = tmpPDF
	d["IsGuest"] = false
	d["MaxGuestSKUs"] = maxGuestSKUs
	d["SavedReportID"] = report.ID
	d["ReportTitle"] = report.Title
	s.render(w, "results.html", d)
}

func (s *Server) handleReportDelete(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	if claims == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if s.db == nil {
		http.Redirect(w, r, "/upload", http.StatusSeeOther)
		return
	}

	reportID := r.PathValue("id")
	_ = s.db.DeleteReport(r.Context(), claims.UserID, reportID)
	http.Redirect(w, r, "/reports", http.StatusSeeOther)
}

// ---------------------------------------------------------------------------
// Rendering helpers
// ---------------------------------------------------------------------------

func (s *Server) render(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (s *Server) renderError(w http.ResponseWriter, r *http.Request, msg string) {
	d := s.baseData(r)
	d["Message"] = msg
	s.render(w, "error.html", d)
}
