package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test that different API keys (same client) get independent rate limits
func TestAPIKeyIsolation(t *testing.T) {
	l := New(map[string]int{"admin": 2}, 100)
	// Two different "clients" (representing different API keys)
	l.Allow("key-1", []string{"admin"})
	l.Allow("key-1", []string{"admin"})
	// key-1 exhausted
	if l.Allow("key-1", []string{"admin"}) {
		t.Error("key-1 should be rate limited")
	}
	// key-2 should still work
	if !l.Allow("key-2", []string{"admin"}) {
		t.Error("key-2 should not be affected by key-1's limit")
	}
}

// Test middleware extracts API key as client identifier
func TestMiddleware_APIKeyExtraction(t *testing.T) {
	l := New(nil, 1) // 1 RPM
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := l.Middleware(inner, func(r *http.Request) (string, []string) {
		if key := r.Header.Get("X-API-Key"); key != "" {
			return "apikey:" + key, []string{"read:logs-*"}
		}
		return "", nil
	})

	// First request passes
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("X-API-Key", "oak_test123")
	handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("first request: expected 200, got %d", w.Code)
	}

	// Second request should be rate limited
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("X-API-Key", "oak_test123")
	handler.ServeHTTP(w, r)
	if w.Code != 429 {
		t.Errorf("second request: expected 429, got %d", w.Code)
	}
}

// Test scope-based rate differentiation
func TestScopeBasedRates(t *testing.T) {
	l := New(map[string]int{"admin": 100, "read:logs-*": 5}, 10)
	// Admin scope gets 100 RPM (most restrictive = min of matching)
	// Actually resolveRPM picks the MOST restrictive (lowest)
	// admin=100, read:logs-*=5 → picks 5
	for i := 0; i < 5; i++ {
		if !l.Allow("client-both", []string{"admin", "read:logs-*"}) {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if l.Allow("client-both", []string{"admin", "read:logs-*"}) {
		t.Error("should be limited at 5 RPM (most restrictive scope)")
	}
}

// Test RetryAfter returns sensible value
func TestRetryAfterValue(t *testing.T) {
	l := New(nil, 1)
	l.Allow("client-x", nil)
	l.Allow("client-x", nil) // over limit
	ra := l.RetryAfter("client-x")
	if ra < 1 || ra > 61 {
		t.Errorf("expected retry-after 1-61, got %d", ra)
	}
}

// Test unknown client has minimal retry-after
func TestRetryAfterUnknownClient(t *testing.T) {
	l := New(nil, 100)
	ra := l.RetryAfter("unknown")
	if ra > 60 {
		t.Errorf("expected retry-after <=60 for fresh client, got %d", ra)
	}
}
