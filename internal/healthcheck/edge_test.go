package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRecovery(t *testing.T) {
	var healthy atomic.Bool
	healthy.Store(false)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if healthy.Load() {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(500)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, 50*time.Millisecond, nil)
	defer c.Stop()
	time.Sleep(80 * time.Millisecond)
	if c.Status().Healthy {
		t.Fatal("should be unhealthy initially")
	}

	healthy.Store(true)
	time.Sleep(80 * time.Millisecond)
	if !c.Status().Healthy {
		t.Fatal("should recover after upstream heals")
	}
}

func TestLastCheckUpdated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := New(srv.URL, 50*time.Millisecond, nil)
	defer c.Stop()
	time.Sleep(80 * time.Millisecond)
	t1 := c.Status().LastCheck
	time.Sleep(80 * time.Millisecond)
	t2 := c.Status().LastCheck
	if !t2.After(t1) {
		t.Fatal("LastCheck should advance between checks")
	}
}

func TestStopHaltsChecks(t *testing.T) {
	var count atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := New(srv.URL, 50*time.Millisecond, nil)
	time.Sleep(80 * time.Millisecond)
	c.Stop()
	after := count.Load()
	time.Sleep(150 * time.Millisecond)
	if count.Load() > after+1 {
		t.Fatal("checks should stop after Stop()")
	}
}

func TestCustomTransport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") == "" {
			w.WriteHeader(200)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, 50*time.Millisecond, http.DefaultTransport)
	defer c.Stop()
	time.Sleep(80 * time.Millisecond)
	if !c.Status().Healthy {
		t.Fatal("should work with custom transport")
	}
}
