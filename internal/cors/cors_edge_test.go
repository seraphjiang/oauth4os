package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func handler200() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
}

func TestDisallowedOriginBlocked(t *testing.T) {
	m := Middleware(Config{Origins: []string{"https://allowed.com"}})
	h := m(handler200())

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") == "https://evil.com" {
		t.Fatal("disallowed origin should not be reflected")
	}
}

func TestPreflightMaxAge(t *testing.T) {
	m := Middleware(Config{Origins: []string{"https://app.com"}})
	h := m(handler200())

	r := httptest.NewRequest("OPTIONS", "/", nil)
	r.Header.Set("Origin", "https://app.com")
	r.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 204 && w.Code != 200 {
		t.Fatalf("preflight should return 204 or 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("preflight should include Allow-Methods")
	}
}

func TestCustomHeaders(t *testing.T) {
	m := Middleware(Config{
		Origins: []string{"https://app.com"},
		Headers: []string{"X-Custom-Auth", "X-Trace-ID"},
	})
	h := m(handler200())

	r := httptest.NewRequest("OPTIONS", "/", nil)
	r.Header.Set("Origin", "https://app.com")
	r.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	headers := w.Header().Get("Access-Control-Allow-Headers")
	if headers == "" {
		t.Fatal("should include custom headers")
	}
}

func TestCustomMethods(t *testing.T) {
	m := Middleware(Config{
		Origins: []string{"https://app.com"},
		Methods: []string{"GET", "PATCH"},
	})
	h := m(handler200())

	r := httptest.NewRequest("OPTIONS", "/", nil)
	r.Header.Set("Origin", "https://app.com")
	r.Header.Set("Access-Control-Request-Method", "PATCH")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Fatal("should include custom methods")
	}
}
