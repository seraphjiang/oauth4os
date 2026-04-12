package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mutation: remove allow check → exceeded limit must be rejected
func TestMutation_ExceededLimit(t *testing.T) {
	l := New(map[string]int{"app": 2}, 60)
	l.Allow("app", nil)
	l.Allow("app", nil)
	if l.Allow("app", nil) {
		t.Error("third request should be rate limited (RPM=2)")
	}
}

// Mutation: remove 429 status → middleware must return 429 when limited
func TestMutation_429Status(t *testing.T) {
	l := New(map[string]int{"app": 1}, 60)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := l.Middleware(inner, func(r *http.Request) (string, []string) { return "app", nil })

	// Exhaust
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	// Should be limited
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 429 {
		t.Errorf("rate limited request should return 429, got %d", w.Code)
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
	if w.Code == 429 && w.Header().Get("Retry-After") == "" {
		t.Error("rate limited response must include Retry-After")
	}
}

// Mutation: remove per-client isolation → different clients must have separate limits
func TestMutation_ClientIsolation(t *testing.T) {
	l := New(map[string]int{"app": 1}, 60)
	l.Allow("app", nil) // exhaust app's limit
	if !l.Allow("other", nil) {
		t.Error("different client should have its own limit")
	}
}
