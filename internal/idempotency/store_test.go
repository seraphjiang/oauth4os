package idempotency

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIdempotencyDedup(t *testing.T) {
	s := New(5 * time.Second)
	defer s.Stop()

	calls := 0
	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(201)
		w.Write([]byte(`{"id":"abc"}`))
	}))

	// First request
	req := httptest.NewRequest("POST", "/", strings.NewReader("{}"))
	req.Header.Set("Idempotency-Key", "key-1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 201 || calls != 1 {
		t.Fatalf("first: got %d, calls=%d", w.Code, calls)
	}

	// Replay — should return cached response without calling handler
	req2 := httptest.NewRequest("POST", "/", strings.NewReader("{}"))
	req2.Header.Set("Idempotency-Key", "key-1")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != 201 || calls != 1 {
		t.Fatalf("replay: got %d, calls=%d (expected 1)", w2.Code, calls)
	}

	// Different key — should call handler
	req3 := httptest.NewRequest("POST", "/", strings.NewReader("{}"))
	req3.Header.Set("Idempotency-Key", "key-2")
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)
	if calls != 2 {
		t.Fatalf("different key: calls=%d (expected 2)", calls)
	}
}

func TestIdempotencySkipsGET(t *testing.T) {
	s := New(5 * time.Second)
	defer s.Stop()

	calls := 0
	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Idempotency-Key", "key-1")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if calls != 2 {
		t.Fatalf("GET should not be deduplicated, calls=%d", calls)
	}
}

func TestStopHaltsReaper(t *testing.T) {
	s := New(10 * time.Millisecond)
	s.Stop()
	// Should not panic or hang
}
