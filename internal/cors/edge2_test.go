package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEdge_EmptyOriginsAllowsAll(t *testing.T) {
	wrap := Middleware(Config{Origins: nil})
	h := wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "https://any-origin.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("empty origins config should allow all")
	}
}

func TestEdge_OptionsReturns204(t *testing.T) {
	wrap := Middleware(Config{Origins: []string{"https://example.com"}})
	h := wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	r := httptest.NewRequest("OPTIONS", "/", nil)
	r.Header.Set("Origin", "https://example.com")
	r.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// Preflight should return 204 or 200
	if w.Code != 204 && w.Code != 200 {
		t.Errorf("preflight should return 204 or 200, got %d", w.Code)
	}
}

func TestEdge_AllowedMethodsHeader(t *testing.T) {
	wrap := Middleware(Config{Origins: []string{"https://example.com"}})
	h := wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	r := httptest.NewRequest("OPTIONS", "/", nil)
	r.Header.Set("Origin", "https://example.com")
	r.Header.Set("Access-Control-Request-Method", "DELETE")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("preflight should include Allow-Methods header")
	}
}
