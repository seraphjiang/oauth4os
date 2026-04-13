package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

// Edge: burst after exhaustion — tokens refill over time
func TestEdge_BurstAfterExhaustion(t *testing.T) {
	l := New(nil, 60) // 60 RPM = 1/sec, burst=60
	scopes := []string{"read"}
	// Exhaust all tokens
	for i := 0; i < 60; i++ {
		l.Allow("client-1", scopes)
	}
	if l.Allow("client-1", scopes) {
		t.Error("should be rate limited after exhausting burst")
	}
	// RetryAfter should be > 0
	ra := l.RetryAfter("client-1")
	if ra <= 0 {
		t.Errorf("RetryAfter should be > 0, got %d", ra)
	}
}

// Edge: concurrent clients hitting limit simultaneously
func TestEdge_ConcurrentClientsLimit(t *testing.T) {
	l := New(nil, 60)
	scopes := []string{"read"}
	var wg sync.WaitGroup
	var allowed atomic.Int64
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.Allow("shared-client", scopes) {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	got := allowed.Load()
	if got > 60 {
		t.Errorf("should allow at most 60 (burst), got %d", got)
	}
	if got == 0 {
		t.Error("should allow at least some requests")
	}
}

// Edge: middleware returns 429 with Retry-After header
func TestEdge_Middleware429Headers(t *testing.T) {
	l := New(nil, 1) // 1 RPM
	extract := func(r *http.Request) (string, []string) {
		return "test-client", []string{"read"}
	}
	handler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), extract)

	// First request should pass
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Errorf("first request should pass, got %d", w.Code)
	}

	// Second request should be rate limited
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 429 {
		t.Errorf("second request should be 429, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("429 response must include Retry-After header")
	}
	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("429 response must include X-RateLimit-Limit header")
	}
}

// Edge: middleware skips rate limiting when extractClient returns empty
func TestEdge_MiddlewareSkipsEmpty(t *testing.T) {
	l := New(nil, 1)
	extract := func(r *http.Request) (string, []string) { return "", nil }
	handler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), extract)

	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 {
			t.Errorf("request %d should pass when client is empty, got %d", i, w.Code)
		}
	}
}

// Edge: per-scope limits override default
func TestEdge_PerScopeLimit(t *testing.T) {
	l := New(map[string]int{"admin": 2}, 600)
	scopes := []string{"admin"}
	// Burst for admin scope = 2
	l.Allow("c1", scopes)
	l.Allow("c1", scopes)
	if l.Allow("c1", scopes) {
		t.Error("admin scope should be limited to 2 RPM burst")
	}
}
