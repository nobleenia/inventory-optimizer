package web

import (
"encoding/json"
"fmt"
"net/http"
"strings"

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
claims := s.currentUser(r)
if claims == nil {
http.Redirect(w, r, "/login?redirect=/records", http.StatusSeeOther)
return
}

d := s.baseData(r)
if err := s.tmpl.ExecuteTemplate(w, "records.html", d); err != nil {
http.Error(w, err.Error(), http.StatusInternalServerError)
}
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

// 4. Send directly as file download
filename := strings.ReplaceAll(tmpl.Name, " ", "_") + ".xlsx"
w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

if err := f.Write(w); err != nil {
http.Error(w, err.Error(), http.StatusInternalServerError)
}
}
