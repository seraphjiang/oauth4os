package loadshed

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConcurrentShedding(t *testing.T) {
	s := New(2) // only 2 concurrent
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
	})
	handler := s.Middleware(inner)

	var wg sync.WaitGroup
	codes := make([]int, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
			codes[n] = w.Code
		}(i)
	}
	wg.Wait()

	got200, got503 := 0, 0
	for _, c := range codes {
		if c == 200 {
			got200++
		} else if c == 503 {
			got503++
		}
	}
	if got503 == 0 {
		t.Error("expected some 503s with 5 concurrent requests and threshold=2")
	}
	if got200 == 0 {
		t.Error("expected some 200s")
	}
}

func TestShedding503Body(t *testing.T) {
	s := New(0) // threshold 0 = reject everything
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := s.Middleware(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	if w.Code != 503 {
		t.Fatalf("expected 503, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "overloaded") {
		t.Error("503 body should contain 'overloaded'")
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("missing Retry-After header")
	}
}

func TestStats(t *testing.T) {
	s := New(0)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := s.Middleware(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	_, rejected := s.Stats()
	if rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", rejected)
	}
}
