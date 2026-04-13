package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Edge: preflight OPTIONS returns CORS headers
func TestEdge_PreflightOptions(t *testing.T) {
	h := New([]string{"https://example.com"})
	handler := h.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	r := httptest.NewRequest("OPTIONS", "/api/resource", nil)
	r.Header.Set("Origin", "https://example.com")
	r.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("preflight must return Access-Control-Allow-Origin")
	}
}

// Edge: disallowed origin gets no CORS headers
func TestEdge_DisallowedOrigin(t *testing.T) {
	h := New([]string{"https://example.com"})
	handler := h.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	r := httptest.NewRequest("GET", "/api/resource", nil)
	r.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") == "https://evil.com" {
		t.Error("disallowed origin should not get CORS header")
	}
}

// Edge: no origin header passes through
func TestEdge_NoOriginPassthrough(t *testing.T) {
	h := New([]string{"https://example.com"})
	handler := h.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	r := httptest.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("no origin should pass through, got %d", w.Code)
	}
}
