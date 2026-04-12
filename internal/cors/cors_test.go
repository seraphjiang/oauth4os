package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAllowAll(t *testing.T) {
	h := Middleware(Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestAllowSpecific(t *testing.T) {
	h := Middleware(Config{Origins: []string{"https://app.example.com"}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))

	// Allowed origin
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("expected specific origin, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}

	// Disallowed origin
	r2 := httptest.NewRequest("GET", "/", nil)
	r2.Header.Set("Origin", "https://evil.com")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no CORS header for disallowed origin, got %s", w2.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestPreflight(t *testing.T) {
	h := Middleware(Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("handler should not be called for preflight") }))
	r := httptest.NewRequest("OPTIONS", "/", nil)
	r.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestNoOriginHeader(t *testing.T) {
	called := false
	h := Middleware(Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !called {
		t.Fatal("handler should be called")
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("no CORS headers expected without Origin")
	}
}
