package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mutation: remove allow check → exceeded limit must be rejected
func TestMutation_ExceededLimit(t *testing.T) {
	l := New(map[string]int{"app": 1}, 60)
	l.Allow("app", nil) // use the 1 allowed
	if l.Allow("app", nil) {
		t.Error("second request should be rate limited")
	}
}

// Mutation: remove default RPM → unknown client must use default
func TestMutation_DefaultRPM(t *testing.T) {
	l := New(nil, 1)
	l.Allow("unknown", nil)
	if l.Allow("unknown", nil) {
		t.Error("default RPM=1 should limit second request")
	}
}

// Mutation: remove 429 status → middleware must return 429 when limited
func TestMutation_429Status(t *testing.T) {
	l := New(map[string]int{"app": 1}, 60)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := l.Middleware(inner, func(r *http.Request) (string, []string) { return "app", nil })

	// First request passes
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Fatalf("first request should pass, got %d", w.Code)
	}

	// Second request limited
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, httptest.NewRequest("GET", "/", nil))
	if w2.Code != 429 {
		t.Errorf("rate limited request should return 429, got %d", w2.Code)
	}
}

// Mutation: remove Retry-After → limited response must include Retry-After
func TestMutation_RetryAfterHeader(t *testing.T) {
	l := New(map[string]int{"app": 1}, 60)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := l.Middleware(inner, func(r *http.Request) (string, []string) { return "app", nil })

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Header().Get("Retry-After") == "" {
		t.Error("rate limited response must include Retry-After")
	}
}
