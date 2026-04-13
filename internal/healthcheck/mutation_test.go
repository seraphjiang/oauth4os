package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Mutation: remove Healthy=true for <500 → healthy upstream must report healthy
func TestMutation_HealthyStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	c := New(srv.URL, 100*time.Millisecond, nil)
	defer c.Stop()
	time.Sleep(150 * time.Millisecond)
	s := c.Status()
	if !s.Healthy {
		t.Error("200 upstream must be healthy")
	}
}

// Mutation: remove error string on 5xx → unhealthy must have error
func TestMutation_UnhealthyError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()
	c := New(srv.URL, 100*time.Millisecond, nil)
	defer c.Stop()
	time.Sleep(150 * time.Millisecond)
	s := c.Status()
	if s.Healthy {
		t.Error("503 upstream must be unhealthy")
	}
	if s.Error == "" {
		t.Error("unhealthy status must include error string")
	}
}

// Mutation: remove latency tracking → latency must be > 0
func TestMutation_LatencyTracked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer srv.Close()
	c := New(srv.URL, 100*time.Millisecond, nil)
	defer c.Stop()
	time.Sleep(150 * time.Millisecond)
	s := c.Status()
	if s.Latency == 0 {
		t.Error("latency must be tracked")
	}
}

// Mutation: remove LastCheck → must record check time
func TestMutation_LastCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	before := time.Now()
	c := New(srv.URL, 100*time.Millisecond, nil)
	defer c.Stop()
	time.Sleep(150 * time.Millisecond)
	s := c.Status()
	if s.LastCheck.Before(before) {
		t.Error("LastCheck must be after checker creation")
	}
}

// Mutation: remove Stop → checker goroutine must terminate
func TestMutation_StopTerminates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	c := New(srv.URL, 50*time.Millisecond, nil)
	done := make(chan struct{})
	go func() {
		c.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop must terminate the checker goroutine")
	}
}

// Mutation: remove zero-interval guard → New with 0 interval must not panic
func TestMutation_ZeroIntervalNoPanic(t *testing.T) {
	c := New("http://localhost:1", 0, nil)
	time.Sleep(50 * time.Millisecond)
	c.Stop()
}
