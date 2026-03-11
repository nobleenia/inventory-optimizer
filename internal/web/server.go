// Package web provides an HTTP server that wraps the inventory
// optimization engine behind a browser-friendly interface.
//
// Users upload two CSV files, the engine runs, and results are rendered
// as an HTML report with options to download CSV or PDF.
package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/noble-ch/inventory-optimizer/internal/engine"
	"github.com/noble-ch/inventory-optimizer/internal/models"
	"github.com/noble-ch/inventory-optimizer/internal/reporting"
)

//go:embed templates/*.html static/css/*.css static/js/*.js
var content embed.FS

// templateFuncs provides custom functions available in HTML templates.
var templateFuncs = template.FuncMap{
	"ceil":     func(f float64) float64 { return math.Ceil(f) },
	"pct":      func(f float64) string { return fmt.Sprintf("%.1f%%", f*100) },
	"currency": func(f float64) string { return fmt.Sprintf("€%.2f", f) },
	"fixed1":   func(f float64) string { return fmt.Sprintf("%.1f", f) },
	"fixed0":   func(f float64) string { return fmt.Sprintf("%.0f", f) },
	"upper":    strings.ToUpper,
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

// Server holds the HTTP server configuration.
type Server struct {
	Addr string
	mux  *http.ServeMux
	tmpl *template.Template
}

// NewServer creates a configured server ready to ListenAndServe.
func NewServer(addr string) *Server {
	s := &Server{
		Addr: addr,
		mux:  http.NewServeMux(),
	}

	// Parse embedded templates.
	s.tmpl = template.Must(
		template.New("").Funcs(templateFuncs).ParseFS(content, "templates/*.html"),
	)

	// Routes.
	s.mux.HandleFunc("/", s.handleLanding)
	s.mux.HandleFunc("/upload", s.handleUpload)
	s.mux.HandleFunc("/analyze", s.handleAnalyze)
	s.mux.HandleFunc("/download/csv", s.handleDownloadCSV)
	s.mux.HandleFunc("/download/pdf", s.handleDownloadPDF)
	s.mux.Handle("/static/", http.FileServer(http.FS(content)))

	return s
}

// Start begins listening. It blocks until the server shuts down.
func (s *Server) Start() error {
	log.Printf("Starting web server on %s …\n", s.Addr)
	log.Printf("Open http://localhost%s in your browser.\n", s.Addr)
	return http.ListenAndServe(s.Addr, s.mux)
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// handleLanding renders the landing / home page.
func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.render(w, "landing.html", map[string]interface{}{
		"Version": models.Version,
	})
}

// handleUpload renders the CSV upload form.
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	s.render(w, "index.html", map[string]interface{}{
		"Version": models.Version,
	})
}

// handleAnalyze processes uploaded CSVs and renders the results page.
func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/upload", http.StatusSeeOther)
		return
	}

	// Limit upload size to 10 MB.
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		s.renderError(w, "Upload too large. Maximum file size is 10 MB.")
		return
	}

	salesFile, _, err := r.FormFile("sales_file")
	if err != nil {
		s.renderError(w, "Please upload a sales history CSV file.")
		return
	}
	defer salesFile.Close()

	paramsFile, _, err := r.FormFile("params_file")
	if err != nil {
		s.renderError(w, "Please upload a SKU parameters CSV file.")
		return
	}
	defer paramsFile.Close()

	opts := engine.DefaultOptions()
	start := time.Now()
	reports, warnings, err := engine.RunFromReaders(salesFile, paramsFile, opts)
	elapsed := time.Since(start)

	if err != nil {
		s.renderError(w, fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	// Store reports in a temp CSV for download.
	tmpFile := fmt.Sprintf("/tmp/inventory-report-%d.csv", time.Now().UnixNano())
	if exportErr := reporting.ExportCSV(tmpFile, reports); exportErr != nil {
		log.Printf("CSV export warning: %v", exportErr)
	}

	// Store reports in a temp PDF for download.
	tmpPDF := fmt.Sprintf("/tmp/inventory-report-%d.pdf", time.Now().UnixNano())
	if exportErr := reporting.ExportPDF(tmpPDF, reports); exportErr != nil {
		log.Printf("PDF export warning: %v", exportErr)
	}

	data := map[string]interface{}{
		"Reports":  reports,
		"Warnings": warnings,
		"Elapsed":  elapsed.Round(time.Millisecond).String(),
		"Version":  models.Version,
		"CSVPath":  tmpFile,
		"PDFPath":  tmpPDF,
		"SKUCount": len(reports),
	}

	s.render(w, "results.html", data)
}

// handleDownloadCSV serves the generated CSV report.
func (s *Server) handleDownloadCSV(w http.ResponseWriter, r *http.Request) {
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

// handleDownloadPDF serves the generated PDF report.
func (s *Server) handleDownloadPDF(w http.ResponseWriter, r *http.Request) {
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
// Rendering helpers
// ---------------------------------------------------------------------------

func (s *Server) render(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (s *Server) renderError(w http.ResponseWriter, msg string) {
	s.render(w, "error.html", map[string]string{"Message": msg})
}
