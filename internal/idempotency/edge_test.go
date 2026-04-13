package idempotency

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Edge: replay returns cached response with X-Idempotent-Replay header
func TestEdge_ReplayHeader(t *testing.T) {
	s := New(time.Minute)
	defer s.Stop()
	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write([]byte(`{"id":"123"}`))
	}))

	// First request
	r := httptest.NewRequest("POST", "/api/resource", strings.NewReader("body"))
	r.Header.Set("Idempotency-Key", "key-1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatalf("first request should return 201, got %d", w.Code)
	}

	// Replay
	r = httptest.NewRequest("POST", "/api/resource", strings.NewReader("body"))
	r.Header.Set("Idempotency-Key", "key-1")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Errorf("replay should return cached 201, got %d", w.Code)
	}
	if w.Header().Get("X-Idempotent-Replay") != "true" {
		t.Error("replay must set X-Idempotent-Replay: true")
	}
}

// Edge: GET requests bypass idempotency
func TestEdge_GETBypasses(t *testing.T) {
	s := New(time.Minute)
	defer s.Stop()
	calls := 0
	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(200)
	}))

	for i := 0; i < 3; i++ {
		r := httptest.NewRequest("GET", "/api/resource", nil)
		r.Header.Set("Idempotency-Key", "key-1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
	if calls != 3 {
		t.Errorf("GET should bypass idempotency, expected 3 calls got %d", calls)
	}
}

// Edge: no Idempotency-Key header bypasses
func TestEdge_NoKeyBypasses(t *testing.T) {
	s := New(time.Minute)
	defer s.Stop()
	calls := 0
	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(200)
	}))

	for i := 0; i < 3; i++ {
		r := httptest.NewRequest("POST", "/api/resource", strings.NewReader("body"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
	if calls != 3 {
		t.Errorf("no key should bypass, expected 3 calls got %d", calls)
	}
}

// Edge: different keys get different responses
func TestEdge_DifferentKeys(t *testing.T) {
	s := New(time.Minute)
	defer s.Stop()
	calls := 0
	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(200)
	}))

	for i := 0; i < 3; i++ {
		r := httptest.NewRequest("POST", "/api/resource", strings.NewReader("body"))
		r.Header.Set("Idempotency-Key", string(rune('a'+i)))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
	if calls != 3 {
		t.Errorf("different keys should each execute, expected 3 calls got %d", calls)
	}
}
