package loadshed

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mutation: change > to >= in threshold check → should still shed at threshold
func TestMutation_ThresholdBoundary(t *testing.T) {
	s := New(1)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := s.Middleware(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Errorf("single request at threshold=1 should pass, got %d", w.Code)
	}
}

// Mutation: remove rejected counter increment → Stats must track rejections
func TestMutation_RejectedCounter(t *testing.T) {
	s := New(0)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := s.Middleware(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	_, rejected := s.Stats()
	if rejected < 1 {
		t.Error("rejected counter must increment on shed")
	}
}

// Mutation: remove defer active.Add(-1) → active count leaks
func TestMutation_ActiveDecrement(t *testing.T) {
	s := New(100)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := s.Middleware(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	active, _ := s.Stats()
	if active != 0 {
		t.Errorf("active should be 0 after request completes, got %d", active)
	}
}

// Mutation: remove 503 status → must return 503 not 200
func TestMutation_503Status(t *testing.T) {
	s := New(0)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := s.Middleware(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 503 {
		t.Errorf("shed must return 503, got %d", w.Code)
	}
}

// Mutation: remove Retry-After header
func TestMutation_RetryAfter(t *testing.T) {
	s := New(0)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := s.Middleware(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Header().Get("Retry-After") == "" {
		t.Error("shed response must include Retry-After header")
	}
}

// Mutation: remove Stats → must return active and rejected counts
func TestMutation_StatsAccurate(t *testing.T) {
	s := New(1)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(200)
	})
	handler := s.Middleware(inner)

	// Start a request that holds the slot
	go func() {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	}()
	time.Sleep(10 * time.Millisecond)

	// This one should be rejected
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	time.Sleep(60 * time.Millisecond) // wait for first to finish
	_, rejected := s.Stats()
	if rejected == 0 {
		t.Error("Stats must report rejected requests")
	}
}
