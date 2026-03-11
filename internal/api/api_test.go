package api

import (
"net/http"
"net/http/httptest"
"testing"
)

func TestWriteJSON(t *testing.T) {
rec := httptest.NewRecorder()
writeJSON(rec, http.StatusOK, map[string]string{"hello": "world"})

if rec.Code != http.StatusOK {
t.Errorf("expected 200, got %d", rec.Code)
}
ct := rec.Header().Get("Content-Type")
if ct != "application/json; charset=utf-8" {
t.Errorf("expected application/json, got %s", ct)
}
body := rec.Body.String()
if body == "" {
t.Fatal("empty body")
}
if !containsStr(body, `"hello":"world"`) {
t.Errorf("body missing data: %s", body)
}
}

func TestWriteError(t *testing.T) {
rec := httptest.NewRecorder()
writeError(rec, http.StatusBadRequest, "bad input")

if rec.Code != http.StatusBadRequest {
t.Errorf("expected 400, got %d", rec.Code)
}
body := rec.Body.String()
if !containsStr(body, `"message":"bad input"`) {
t.Errorf("body missing error message: %s", body)
}
if !containsStr(body, `"code":400`) {
t.Errorf("body missing error code: %s", body)
}
}

func TestWriteJSONWithMeta(t *testing.T) {
rec := httptest.NewRecorder()
writeJSONWithMeta(rec, http.StatusOK, []string{"a", "b"}, &meta{Total: 10, Limit: 2, Offset: 0})

body := rec.Body.String()
if !containsStr(body, `"total":10`) {
t.Errorf("body missing meta total: %s", body)
}
}

func TestCORSMiddleware(t *testing.T) {
handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusOK)
}))

req := httptest.NewRequest(http.MethodGet, "/", nil)
rec := httptest.NewRecorder()
handler.ServeHTTP(rec, req)

if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
t.Error("missing CORS Allow-Origin header")
}

req = httptest.NewRequest(http.MethodOptions, "/", nil)
rec = httptest.NewRecorder()
handler.ServeHTTP(rec, req)

if rec.Code != http.StatusNoContent {
t.Errorf("expected 204 for OPTIONS, got %d", rec.Code)
}
}

func TestLogMiddleware(t *testing.T) {
handler := logMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusCreated)
}))

req := httptest.NewRequest(http.MethodPost, "/test", nil)
rec := httptest.NewRecorder()
handler.ServeHTTP(rec, req)

if rec.Code != http.StatusCreated {
t.Errorf("expected 201, got %d", rec.Code)
}
}

func TestRateLimiter_AllowBurst(t *testing.T) {
rl := NewRateLimiter(5, 5, 1*60*1e9)

for i := 0; i < 5; i++ {
if !rl.Allow("user-1") {
t.Fatalf("request %d should be allowed", i+1)
}
}

if rl.Allow("user-1") {
t.Fatal("6th request should be rate limited")
}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
rl := NewRateLimiter(1, 1, 1*60*1e9)

if !rl.Allow("user-a") {
t.Fatal("first request for user-a should be allowed")
}
if !rl.Allow("user-b") {
t.Fatal("first request for user-b should be allowed")
}
if rl.Allow("user-a") {
t.Fatal("second request for user-a should be denied")
}
}

func containsStr(s, sub string) bool {
return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
for i := 0; i <= len(s)-len(sub); i++ {
if s[i:i+len(sub)] == sub {
return true
}
}
return false
}
