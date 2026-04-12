package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func serve(cfg Config, method, origin string) *httptest.ResponseRecorder {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := Middleware(cfg)(inner)
	r := httptest.NewRequest(method, "/", nil)
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

// Mutation: remove Allow-Origin header → must be set for CORS requests
func TestMutation_AllowOrigin(t *testing.T) {
	w := serve(Config{}, "GET", "https://example.com")
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("must set Access-Control-Allow-Origin")
	}
}

// Mutation: remove OPTIONS 204 → preflight must return 204
func TestMutation_Preflight204(t *testing.T) {
	w := serve(Config{}, "OPTIONS", "https://example.com")
	if w.Code != 204 {
		t.Errorf("preflight should return 204, got %d", w.Code)
	}
}

// Mutation: remove origin check → disallowed origin must get 403 on preflight
func TestMutation_DisallowedOrigin(t *testing.T) {
	w := serve(Config{Origins: []string{"https://allowed.com"}}, "OPTIONS", "https://evil.com")
	if w.Code != 403 {
		t.Errorf("disallowed origin preflight should be 403, got %d", w.Code)
	}
}

// Mutation: remove Max-Age → must set cache header
func TestMutation_MaxAge(t *testing.T) {
	w := serve(Config{}, "GET", "https://example.com")
	if w.Header().Get("Access-Control-Max-Age") != "86400" {
		t.Error("must set Access-Control-Max-Age: 86400")
	}
}

// Mutation: no Origin header → must skip CORS headers entirely
func TestMutation_NoOriginSkips(t *testing.T) {
	w := serve(Config{}, "GET", "")
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("no Origin header should skip CORS headers")
	}
}

// Mutation: remove preflight handling → OPTIONS must return CORS headers
func TestMutation_PreflightHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := Middleware(Config{Origins: []string{"https://app.example.com"}})(inner)
	r := httptest.NewRequest("OPTIONS", "/api/query", nil)
	r.Header.Set("Origin", "https://app.example.com")
	r.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("preflight must set Access-Control-Allow-Origin")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("preflight must set Access-Control-Allow-Methods")
	}
}
