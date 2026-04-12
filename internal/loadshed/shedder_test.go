package loadshed

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestAllowUnderCapacity(t *testing.T) {
	s := New(10)
	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestShedOverCapacity(t *testing.T) {
	s := New(1)
	block := make(chan struct{})
	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
		w.WriteHeader(200)
	}))

	// Fill capacity
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	}()

	// Wait for first request to be inflight
	for s.inflight.Load() == 0 {
	}

	// Second request should be shed
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 503 {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	_, shed := s.Stats()
	if shed != 1 {
		t.Fatalf("expected 1 shed, got %d", shed)
	}

	close(block)
	wg.Wait()
}
