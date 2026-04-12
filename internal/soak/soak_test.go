package soak

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMutation_NoLeak(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	r := Run(srv.URL, 2*time.Second, 4)
	t.Logf("Soak: %s", r)
	if r.Requests == 0 {
		t.Error("soak test must execute requests")
	}
	if r.Leaked {
		t.Errorf("memory leak detected: heap +%.1fMB, goroutines +%d", r.HeapGrowth, r.GRGrowth)
	}
}

func TestMutation_RequestsCounted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	r := Run(srv.URL, 500*time.Millisecond, 2)
	if r.Requests < 10 {
		t.Errorf("expected many requests in 500ms, got %d", r.Requests)
	}
	if r.Duration != 500*time.Millisecond {
		t.Errorf("expected 500ms duration, got %s", r.Duration)
	}
}
