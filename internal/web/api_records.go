package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/noble-ch/inventory-optimizer/internal/records"
	"github.com/noble-ch/inventory-optimizer/internal/store"
)

// HandleGetTemplates returns the list of available Excel templates.
func (s *Server) HandleGetTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	templates := records.GetAvailableTemplates()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// handleRecordsPage renders the Smart Records UI for premium users.
func (s *Server) handleRecordsPage(w http.ResponseWriter, r *http.Request) {
	s.serveApp(w)
}

type generateRequest struct {
	TemplateID string   `json:"template_id"`
	Prefill    bool     `json:"prefill"`
	Columns    []string `json:"columns"`
}

// HandleGenerateRecord handles the form submission to build the Excel file dynamically.
func (s *Server) HandleGenerateRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := s.currentUser(r)
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// 1. Locate the Template
	var tmpl *records.Template
	for _, t := range records.GetAvailableTemplates() {
		if t.ID == req.TemplateID {
			tmpl = &t
			break
		}
	}
	if tmpl == nil {
		http.Error(w, "Template not found", http.StatusBadRequest)
		return
	}

	// 2. Fetch User SKUs if Prefill selected
	var skus []store.SKU
	if req.Prefill && s.db != nil {
		userSkus, err := s.db.GetSKUs(r.Context(), claims.Subject)
		if err == nil {
			skus = userSkus
		}
	}

	// 3. Generate the Excel File
	f, err := records.GenerateExcel(*tmpl, req.Columns, skus)
	if err != nil {
		http.Error(w, "Failed to generate Excel: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Save to disk as history/archive
	if err := os.MkdirAll("data/generated", 0755); err == nil {
		timestamp := time.Now().Format("20060102_150405")
		safeName := strings.ReplaceAll(tmpl.Name, " ", "_")
		localPath := fmt.Sprintf("data/generated/%s_%s_%s.xlsx", claims.Subject, safeName, timestamp)
		_ = f.SaveAs(localPath) // We ignore errors for disk saves so it doesn't block the user download

		// Log into database
		if s.db != nil {
			err = s.db.SaveGeneratedRecord(r.Context(), &store.GeneratedRecord{
				UserID:       claims.Subject,
				TemplateName: tmpl.Name,
				FilePath:     localPath,
				RecordsCount: len(skus),
			})
			if err != nil {
				fmt.Println("Warning: Failed to save record history metadata:", err)
			}
		}
	}

	// 5. Send directly as file download
	filename := strings.ReplaceAll(tmpl.Name, " ", "_") + ".xlsx"
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	if err := f.Write(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HandleGetRecordsHistory fetches the previously generated Excel targets for a premium user.
func (s *Server) HandleGetRecordsHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := s.currentUser(r)
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if s.db == nil {
		// Provide an empty mock if no database
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal([]store.GeneratedRecord{})
		w.Write(b)
		return
	}

	records, err := s.db.GetGeneratedRecords(r.Context(), claims.Subject)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// HandleDownloadRecord fetches a previously generated file from history and serves its binary block.
func (s *Server) HandleDownloadRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := s.currentUser(r)
	if claims == nil {
		http.Redirect(w, r, "/login?redirect=/records", http.StatusSeeOther)
		return
	}

	recordID := r.PathValue("id")
	if recordID == "" {
		http.Error(w, "Missing record ID", http.StatusBadRequest)
		return
	}

	if s.db == nil {
		http.Error(w, "Database unavailable", http.StatusInternalServerError)
		return
	}

	record, err := s.db.GetGeneratedRecord(r.Context(), claims.Subject, recordID)
	if err != nil {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s_Historical.xlsx\"", strings.ReplaceAll(record.TemplateName, " ", "_")))
	http.ServeFile(w, r, record.FilePath)
}
