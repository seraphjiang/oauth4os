package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEdge_FlappingBackend(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls%2 == 0 {
			w.WriteHeader(500)
		} else {
			w.WriteHeader(200)
		}
	}))
	defer srv.Close()
	c := New(srv.URL, 50*time.Millisecond, nil)
	time.Sleep(300 * time.Millisecond)
	c.Stop()
	s := c.Status()
	// Should have a status — either healthy or not
	_ = s.Healthy
}

func TestEdge_LatencyTracked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer srv.Close()
	c := New(srv.URL, 50*time.Millisecond, nil)
	time.Sleep(150 * time.Millisecond)
	c.Stop()
	s := c.Status()
	if s.Latency == 0 {
		t.Error("latency should be tracked")
	}
}
