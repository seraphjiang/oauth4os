package idempotency

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test that same idempotency key returns cached response
func TestReplayResponse(t *testing.T) {
	s := New(5 * time.Second)
	defer s.Stop()
	callCount := 0
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(201)
		w.Write([]byte(`{"id":"tok_123"}`))
	})
	handler := s.Middleware(inner)

	// First request
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/oauth/token", nil)
	r.Header.Set("Idempotency-Key", "key-1")
	handler.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatalf("first: expected 201, got %d", w.Code)
	}

	// Replay with same key
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/oauth/token", nil)
	r2.Header.Set("Idempotency-Key", "key-1")
	handler.ServeHTTP(w2, r2)
	if w2.Code != 201 {
		t.Fatalf("replay: expected 201, got %d", w2.Code)
	}
	if callCount != 1 {
		t.Errorf("handler should be called once, got %d", callCount)
	}
}

// Test that different keys call handler separately
func TestDifferentKeys(t *testing.T) {
	s := New(5 * time.Second)
	defer s.Stop()
	callCount := 0
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(200)
	})
	handler := s.Middleware(inner)

	for _, key := range []string{"a", "b", "c"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/test", nil)
		r.Header.Set("Idempotency-Key", key)
		handler.ServeHTTP(w, r)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls for 3 different keys, got %d", callCount)
	}
}

// Test that requests without key pass through normally
func TestNoKeyPassThrough(t *testing.T) {
	s := New(5 * time.Second)
	defer s.Stop()
	callCount := 0
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(200)
	})
	handler := s.Middleware(inner)

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("POST", "/test", nil))
	}
	if callCount != 3 {
		t.Errorf("no idempotency key = no caching, expected 3 calls, got %d", callCount)
	}
}

// Test TTL expiry
func TestKeyExpiry(t *testing.T) {
	s := New(50 * time.Millisecond)
	defer s.Stop()
	callCount := 0
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(200)
	})
	handler := s.Middleware(inner)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	r.Header.Set("Idempotency-Key", "expire-me")
	handler.ServeHTTP(w, r)

	time.Sleep(100 * time.Millisecond)

	// After TTL, same key should call handler again
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/test", nil)
	r2.Header.Set("Idempotency-Key", "expire-me")
	handler.ServeHTTP(w2, r2)

	if callCount != 2 {
		t.Errorf("expected 2 calls after TTL expiry, got %d", callCount)
	}
}
