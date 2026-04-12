package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAllow_UnderLimit(t *testing.T) {
	l := New(map[string]int{"read:logs-*": 60}, 600)
	for i := 0; i < 60; i++ {
		if !l.Allow("client1", []string{"read:logs-*"}) {
			t.Fatalf("request %d should be allowed", i)
		}
	}
}

func TestAllow_OverLimit(t *testing.T) {
	l := New(map[string]int{"read:logs-*": 10}, 600)
	// Exhaust burst (capacity = RPM = 10)
	for i := 0; i < 10; i++ {
		l.Allow("client1", []string{"read:logs-*"})
	}
	if l.Allow("client1", []string{"read:logs-*"}) {
		t.Fatal("should be rate limited after burst")
	}
}

func TestAllow_DefaultRPM(t *testing.T) {
	l := New(map[string]int{}, 60)
	for i := 0; i < 60; i++ {
		if !l.Allow("client1", []string{"unknown:scope"}) {
			t.Fatalf("request %d should use default RPM", i)
		}
	}
}

func TestAllow_MostRestrictiveScope(t *testing.T) {
	l := New(map[string]int{"read:logs-*": 100, "admin": 10}, 600)
	// With both scopes, should use admin's 10 RPM
	for i := 0; i < 10; i++ {
		l.Allow("client1", []string{"read:logs-*", "admin"})
	}
	if l.Allow("client1", []string{"read:logs-*", "admin"}) {
		t.Fatal("should use most restrictive scope limit")
	}
}

func TestAllow_PerClientIsolation(t *testing.T) {
	l := New(map[string]int{"read:logs-*": 5}, 600)
	for i := 0; i < 5; i++ {
		l.Allow("client1", []string{"read:logs-*"})
	}
	// client2 should still have its own bucket
	if !l.Allow("client2", []string{"read:logs-*"}) {
		t.Fatal("client2 should have its own bucket")
	}
}

func TestRetryAfter(t *testing.T) {
	l := New(map[string]int{"read:logs-*": 60}, 600)
	for i := 0; i < 60; i++ {
		l.Allow("client1", []string{"read:logs-*"})
	}
	retry := l.RetryAfter("client1")
	if retry < 1 {
		t.Fatalf("retry_after should be >= 1, got %d", retry)
	}
}

func TestMiddleware_Allows(t *testing.T) {
	l := New(map[string]int{}, 600)
	handler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), func(r *http.Request) (string, []string) {
		return "test-client", []string{"read:logs-*"}
	})
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestMiddleware_Returns429(t *testing.T) {
	l := New(map[string]int{"read:logs-*": 1}, 600)
	handler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), func(r *http.Request) (string, []string) {
		return "test-client", []string{"read:logs-*"}
	})
	// First request uses the burst
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Second should be rate limited
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After header")
	}
}

func TestMiddleware_SkipsNoClient(t *testing.T) {
	l := New(map[string]int{}, 1)
	handler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), func(r *http.Request) (string, []string) {
		return "", nil // no client = skip rate limiting
	})
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}
	}
}
