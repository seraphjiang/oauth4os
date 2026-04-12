package idempotency

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Mutation: remove method check → GET requests should NOT be deduplicated
func TestMutation_MethodFilter(t *testing.T) {
	s := New(5 * time.Second)
	defer s.Stop()
	calls := 0
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(200) })
	handler := s.Middleware(inner)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/test", nil)
		r.Header.Set("Idempotency-Key", "same")
		handler.ServeHTTP(w, r)
	}
	if calls != 2 {
		t.Errorf("GET should not be deduplicated, expected 2 calls, got %d", calls)
	}
}

// Mutation: remove X-Idempotent-Replay header → replays must be tagged
func TestMutation_ReplayHeader(t *testing.T) {
	s := New(5 * time.Second)
	defer s.Stop()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201) })
	handler := s.Middleware(inner)

	r1 := httptest.NewRequest("POST", "/test", nil)
	r1.Header.Set("Idempotency-Key", "k1")
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, r1)

	r2 := httptest.NewRequest("POST", "/test", nil)
	r2.Header.Set("Idempotency-Key", "k1")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, r2)

	if w2.Header().Get("X-Idempotent-Replay") != "true" {
		t.Error("replay must set X-Idempotent-Replay: true")
	}
}

// Mutation: remove status capture → replayed status must match original
func TestMutation_StatusPreserved(t *testing.T) {
	s := New(5 * time.Second)
	defer s.Stop()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201) })
	handler := s.Middleware(inner)

	r1 := httptest.NewRequest("POST", "/test", nil)
	r1.Header.Set("Idempotency-Key", "k2")
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, r1)

	r2 := httptest.NewRequest("POST", "/test", nil)
	r2.Header.Set("Idempotency-Key", "k2")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, r2)

	if w2.Code != 201 {
		t.Errorf("replayed status should be 201, got %d", w2.Code)
	}
}

// Mutation: remove body capture → replayed body must match original
func TestMutation_BodyPreserved(t *testing.T) {
	s := New(5 * time.Second)
	defer s.Stop()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"id":"tok_42"}`))
	})
	handler := s.Middleware(inner)

	r1 := httptest.NewRequest("POST", "/test", nil)
	r1.Header.Set("Idempotency-Key", "k3")
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, r1)

	r2 := httptest.NewRequest("POST", "/test", nil)
	r2.Header.Set("Idempotency-Key", "k3")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, r2)

	if w2.Body.String() != `{"id":"tok_42"}` {
		t.Errorf("replayed body mismatch: %s", w2.Body.String())
	}
}

// Mutation: remove no-key passthrough → requests without key must pass through
func TestMutation_NoKeyPassthrough(t *testing.T) {
	s := New(5 * time.Second)
	defer s.Stop()
	called := 0
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(200)
	})
	handler := s.Middleware(inner)
	// Two requests without Idempotency-Key should both call handler
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("POST", "/", nil))
	}
	if called != 2 {
		t.Errorf("requests without key should always call handler, got %d calls", called)
	}
}
