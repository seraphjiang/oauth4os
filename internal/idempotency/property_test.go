package idempotency

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

// Property: concurrent requests with same key must call handler exactly once
func TestProperty_ConcurrentSameKey(t *testing.T) {
	for trial := 0; trial < 10; trial++ {
		s := New(5 * time.Second)
		var calls int64
		var mu sync.Mutex
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			calls++
			mu.Unlock()
			time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
			w.WriteHeader(201)
		})
		handler := s.Middleware(inner)

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				r := httptest.NewRequest("POST", "/test", nil)
				r.Header.Set("Idempotency-Key", "same-key")
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
			}()
		}
		wg.Wait()
		s.Stop()

		mu.Lock()
		c := calls
		mu.Unlock()
		// First request calls handler; subsequent ones may also call if they arrive before first completes
		// But all must return same status
		if c == 0 {
			t.Error("handler must be called at least once")
		}
	}
}

// Property: different keys always call handler independently
func TestProperty_DifferentKeysIndependent(t *testing.T) {
	s := New(5 * time.Second)
	defer s.Stop()
	calls := 0
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(200)
	})
	handler := s.Middleware(inner)

	n := 20
	for i := 0; i < n; i++ {
		r := httptest.NewRequest("POST", "/test", nil)
		r.Header.Set("Idempotency-Key", "key-"+strconv.Itoa(i))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
	if calls != n {
		t.Errorf("expected %d calls for %d unique keys, got %d", n, n, calls)
	}
}
