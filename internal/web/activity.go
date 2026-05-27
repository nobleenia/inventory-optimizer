package web

import (
	"net/http"
	"strconv"

	"github.com/noble-ch/inventory-optimizer/internal/store"
)

func (s *Server) handleAPIActivity(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	if claims == nil {
		s.sendErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if s.db == nil {
		s.sendErrorJSON(w, http.StatusServiceUnavailable, "Database not configured")
		return
	}

	limit := 12
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	events, err := s.db.ListRecentActivity(r.Context(), claims.Subject, limit)
	if err != nil {
		s.sendErrorJSON(w, http.StatusInternalServerError, "Failed to load recent activity")
		return
	}

	if events == nil {
		events = []store.ActivityEvent{}
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{"activity": events})
}
