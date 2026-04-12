package circuit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Mutation: Allow always returns true (breaker never opens)
func TestMutation_NeverOpens(t *testing.T) {
	b := New(2, 1*time.Second)
	b.Record(500)
	b.Record(500)
	if b.Allow() {
		t.Error("MUTATION SURVIVED: breaker should be open after threshold failures")
	}
}

// Mutation: Record ignores 5xx (failures never counted)
func TestMutation_FailuresIgnored(t *testing.T) {
	b := New(1, 1*time.Second)
	b.Record(500)
	if b.Allow() {
		t.Error("MUTATION SURVIVED: single 5xx should trip threshold=1 breaker")
	}
}

// Mutation: Success doesn't reset failure count
func TestMutation_SuccessNoReset(t *testing.T) {
	b := New(3, 1*time.Second)
	b.Record(500)
	b.Record(500)
	b.Record(200) // should reset
	if !b.Allow() {
		t.Error("MUTATION SURVIVED: success should reset failures, breaker should be closed")
	}
	// Two more failures should not trip (reset happened)
	b.Record(500)
	b.Record(500)
	if !b.Allow() {
		t.Error("MUTATION SURVIVED: only 2 failures after reset, threshold is 3")
	}
}

// Mutation: HalfOpen never transitions back to Closed
func TestMutation_HalfOpenStuck(t *testing.T) {
	b := New(1, 50*time.Millisecond)
	b.Record(500) // open
	time.Sleep(100 * time.Millisecond)
	b.Allow()      // transitions to half-open
	b.Record(200)  // should close
	if !b.Allow() {
		t.Error("MUTATION SURVIVED: successful probe should close the breaker")
	}
}

// Mutation: RetryAfter returns 0 when open
func TestMutation_RetryAfterZeroWhenOpen(t *testing.T) {
	b := New(1, 10*time.Second)
	b.Record(500)
	ra := b.RetryAfter()
	if ra == 0 {
		t.Error("MUTATION SURVIVED: RetryAfter should be >0 when open")
	}
}

// Mutation: remove circuit breaker middleware → must reject when open
func TestMutation_MiddlewareRejectsWhenOpen(t *testing.T) {
	b := New(2, 50*time.Millisecond)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := b.Middleware(inner)

	// Trip the breaker
	b.Record(500)
	b.Record(500)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 503 {
		t.Errorf("open breaker should return 503, got %d", w.Code)
	}
}

// Mutation: remove passthrough when closed → must forward when healthy
func TestMutation_MiddlewareForwardsWhenClosed(t *testing.T) {
	b := New(10, time.Second)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	handler := b.Middleware(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 201 {
		t.Errorf("closed breaker should forward, got %d", w.Code)
	}
}
