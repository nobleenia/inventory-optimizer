package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// envelope wraps all JSON responses in a consistent shape.
type envelope struct {
	Data  interface{} `json:"data,omitempty"`
	Error *apiError   `json:"error,omitempty"`
	Meta  *meta       `json:"meta,omitempty"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type meta struct {
	Total  int `json:"total,omitempty"`
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// writeJSON sends a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	resp := envelope{Data: data}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

// writeJSONWithMeta sends a paginated JSON response.
func writeJSONWithMeta(w http.ResponseWriter, status int, data interface{}, m *meta) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	resp := envelope{Data: data, Meta: m}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

// writeError sends a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	resp := envelope{Error: &apiError{Code: status, Message: message}}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

// decodeJSON reads and validates a JSON request body.
func decodeJSON(r *http.Request, dst interface{}) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1 MB max
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
