package loadshed

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// Property: active count always returns to 0 after all requests complete
func TestProperty_ActiveReturnsToZero(t *testing.T) {
	for i := 0; i < 10; i++ {
		threshold := rand.Intn(10) + 1
		s := New(threshold)
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
			w.WriteHeader(200)
		})
		handler := s.Middleware(inner)

		var wg sync.WaitGroup
		n := rand.Intn(20) + 5
		for j := 0; j < n; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
			}()
		}
		wg.Wait()

		active, _ := s.Stats()
		if active != 0 {
			t.Errorf("active=%d after all requests (threshold=%d, n=%d)", active, threshold, n)
		}
	}
}

// Property: rejected + passed = total requests
func TestProperty_CountsAddUp(t *testing.T) {
	s := New(1)
	passed := 0
	var mu sync.Mutex
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		mu.Lock()
		passed++
		mu.Unlock()
		w.WriteHeader(200)
	})
	handler := s.Middleware(inner)

	var wg sync.WaitGroup
	total := 20
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		}()
	}
	wg.Wait()

	_, rejected := s.Stats()
	mu.Lock()
	p := passed
	mu.Unlock()
	if int(rejected)+p != total {
		t.Errorf("rejected(%d) + passed(%d) != total(%d)", rejected, p, total)
	}
}
