package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mutation: remove per-client isolation → different clients must have separate limits
func TestMutation_ClientIsolation(t *testing.T) {
	l := New(nil, 60)
	// Both clients should be allowed independently
	if !l.Allow("client-a", nil) {
		t.Error("client-a should be allowed")
	}
	if !l.Allow("client-b", nil) {
		t.Error("client-b should have its own bucket")
	}
}

// Mutation: remove scope-based RPM → scoped limits must override default
func TestMutation_ScopeBasedRPM(t *testing.T) {
	l := New(map[string]int{"admin": 1000}, 60)
	// Admin scope should get higher limit
	for i := 0; i < 100; i++ {
		if !l.Allow("app", []string{"admin"}) {
			t.Fatalf("admin scope should allow 100 requests, failed at %d", i+1)
		}
	}
}

// Mutation: remove RetryAfter → must return positive value when limited
func TestMutation_RetryAfterPositive(t *testing.T) {
	l := New(map[string]int{"app": 1}, 60)
	// Exhaust bucket
	for l.Allow("app", nil) {
		// drain
	}
	ra := l.RetryAfter("app")
	if ra <= 0 {
		t.Errorf("RetryAfter should be positive when limited, got %d", ra)
	}
}

// Mutation: remove bucket creation → new client must get a bucket
func TestMutation_NewClientBucket(t *testing.T) {
	l := New(nil, 120)
	if !l.Allow("brand-new-client", nil) {
		t.Error("new client should be allowed on first request")
	}
}

// Mutation: remove 429 response → middleware must reject over-limit requests
func TestMutation_MiddlewareRejects(t *testing.T) {
	l := New(nil, 1) // 1 RPM
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := l.Middleware(inner, func(r *http.Request) (string, []string) {
		return "client", nil
	})
	// First request passes
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Fatalf("first request should pass, got %d", w.Code)
	}
	// Exhaust bucket
	for i := 0; i < 10; i++ {
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	}
	if w.Code != 429 {
		t.Errorf("over-limit request should get 429, got %d", w.Code)
	}
}

// Mutation: remove rate limit headers → must set X-RateLimit-Limit
func TestMutation_RateLimitHeaders(t *testing.T) {
	l := New(nil, 600)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := l.Middleware(inner, func(r *http.Request) (string, []string) {
		return "client", nil
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("must set X-RateLimit-Limit header")
	}
	if w.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("must set X-RateLimit-Remaining header")
	}
}
