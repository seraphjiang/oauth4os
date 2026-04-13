package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEdge_HealthyBackend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	c := New(srv.URL, 100*time.Millisecond, nil)
	time.Sleep(200 * time.Millisecond)
	s := c.Status()
	if !s.Healthy {
		t.Error("healthy backend should report healthy")
	}
	c.Stop()
}

func TestEdge_UnhealthyBackend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	c := New(srv.URL, 100*time.Millisecond, nil)
	time.Sleep(200 * time.Millisecond)
	s := c.Status()
	if s.Healthy {
		t.Error("500 backend should report unhealthy")
	}
	c.Stop()
}

func TestEdge_StopIdempotent(t *testing.T) {
	c := New("http://localhost:1", time.Hour, nil)
	c.Stop()
	c.Stop() // must not panic
}
