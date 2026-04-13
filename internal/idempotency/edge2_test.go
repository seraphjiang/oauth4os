package idempotency

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestEdge_ConcurrentSameKey(t *testing.T) {
	s := New(time.Minute)
	defer s.Stop()
	calls := 0
	var mu sync.Mutex
	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		w.WriteHeader(201)
		w.Write([]byte(`{"ok":true}`))
	}))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("POST", "/api", strings.NewReader("body"))
			r.Header.Set("Idempotency-Key", "same-key")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
		}()
	}
	wg.Wait()
}

func TestEdge_PUTSupported(t *testing.T) {
	s := New(time.Minute)
	defer s.Stop()
	calls := 0
	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(200)
	}))

	r := httptest.NewRequest("PUT", "/api/resource", strings.NewReader("body"))
	r.Header.Set("Idempotency-Key", "put-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("PUT should be supported, got %d", w.Code)
	}

	// Replay
	r = httptest.NewRequest("PUT", "/api/resource", strings.NewReader("body"))
	r.Header.Set("Idempotency-Key", "put-key")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if calls != 1 {
		t.Errorf("replay should not call handler again, calls=%d", calls)
	}
}
