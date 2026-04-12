package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := New(srv.URL, 100*time.Millisecond, nil)
	defer c.Stop()
	time.Sleep(200 * time.Millisecond)
	s := c.Status()
	if !s.Healthy {
		t.Errorf("expected healthy, got %+v", s)
	}
	if s.Latency == 0 {
		t.Error("latency should be >0")
	}
}

func TestUnhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	c := New(srv.URL, 100*time.Millisecond, nil)
	defer c.Stop()
	time.Sleep(200 * time.Millisecond)
	s := c.Status()
	if s.Healthy {
		t.Error("expected unhealthy for 503")
	}
}

func TestUnreachable(t *testing.T) {
	c := New("http://127.0.0.1:1", 100*time.Millisecond, nil)
	defer c.Stop()
	time.Sleep(200 * time.Millisecond)
	s := c.Status()
	if s.Healthy {
		t.Error("expected unhealthy for unreachable")
	}
	if s.Error == "" {
		t.Error("expected error message")
	}
}
