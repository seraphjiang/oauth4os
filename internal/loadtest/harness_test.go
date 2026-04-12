package loadtest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func fakeProxy() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"access_token":"tok","token_type":"Bearer","expires_in":3600}`))
	}))
}

// Mutation: remove concurrent execution → must run all clients
func TestMutation_AllClientsRun(t *testing.T) {
	srv := fakeProxy()
	defer srv.Close()
	h := New(srv.URL, 5, 3)
	r := h.Run()
	if r.Total != 15 {
		t.Errorf("expected 15 total (5 clients × 3 iterations), got %d", r.Total)
	}
	if r.Success != 15 {
		t.Errorf("expected 15 success, got %d", r.Success)
	}
}

// Mutation: remove percentile calculation → must compute latency stats
func TestMutation_PercentileComputed(t *testing.T) {
	srv := fakeProxy()
	defer srv.Close()
	h := New(srv.URL, 2, 5)
	r := h.Run()
	if r.P50 == 0 {
		t.Error("P50 must be computed")
	}
	if r.P95 == 0 {
		t.Error("P95 must be computed")
	}
	if r.RPS == 0 {
		t.Error("RPS must be computed")
	}
}

// Mutation: remove error counting → errors must be tracked
func TestMutation_ErrorsCounted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	h := New(srv.URL, 2, 2)
	r := h.Run()
	if r.Errors != 4 {
		t.Errorf("expected 4 errors, got %d", r.Errors)
	}
}

// Property: concurrent harness must not panic
func TestProperty_ConcurrentHarness(t *testing.T) {
	srv := fakeProxy()
	defer srv.Close()
	h := New(srv.URL, 10, 10)
	r := h.Run()
	if r.Total != 100 {
		t.Errorf("expected 100 total, got %d", r.Total)
	}
}
