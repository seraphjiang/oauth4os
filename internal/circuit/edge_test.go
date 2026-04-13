package circuit

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Edge: breaker opens after threshold failures
func TestEdge_OpensAfterThreshold(t *testing.T) {
	b := New(3, 0) // 3 failures to open, no reset
	fail := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	h := b.Wrap(fail)
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	}
	// Next request should be rejected by breaker
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 503 {
		t.Errorf("breaker should return 503 when open, got %d", w.Code)
	}
}

// Edge: successful requests don't trip breaker
func TestEdge_SuccessNoTrip(t *testing.T) {
	b := New(3, 0)
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	h := b.Wrap(ok)
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 {
			t.Errorf("request %d should pass, got %d", i, w.Code)
		}
	}
}

// Edge: concurrent requests through breaker must not panic
func TestEdge_ConcurrentRequests(t *testing.T) {
	b := New(100, 0)
	h := b.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		}()
	}
	wg.Wait()
}
