package loadshed

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

func TestEdge_ConcurrentMiddleware200(t *testing.T) {
	s := New(100)
	h := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	var wg sync.WaitGroup
	var ok atomic.Int64
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
			if w.Code == 200 {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() == 0 {
		t.Error("some requests should pass")
	}
}

func TestEdge_StatsAfterRequests(t *testing.T) {
	s := New(1)
	h := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	}
	_, rejected := s.Stats()
	_ = rejected // just verify no panic
}
