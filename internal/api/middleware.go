package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

// contextKey is a private type to prevent collisions in context values.
type contextKey string

const userIDKey contextKey = "user_id"
const emailKey contextKey = "email"

// requireAuth is middleware that validates a JWT Bearer token and injects
// the user ID and email into the request context.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeError(w, http.StatusUnauthorized, "Missing Authorization header")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			writeError(w, http.StatusUnauthorized, "Authorization header must be: Bearer <token>")
			return
		}

		claims, err := s.auth.ValidateAccessToken(parts[1])
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		ctx = context.WithValue(ctx, emailKey, claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// userFromContext extracts the authenticated user ID from the request context.
func userFromContext(ctx context.Context) (string, string) {
	uid, _ := ctx.Value(userIDKey).(string)
	email, _ := ctx.Value(emailKey).(string)
	return uid, email
}

// corsMiddleware adds permissive CORS headers for API clients.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// logMiddleware logs each request with method, path, status, and duration.
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("API %s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Microsecond))
	})
}

// statusWriter wraps ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// chainMiddleware applies middleware in order (outermost first).
func chainMiddleware(handler http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return handler
}
