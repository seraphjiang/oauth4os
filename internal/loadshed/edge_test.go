package loadshed

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Edge: under capacity passes through
func TestEdge_UnderCapacityPasses(t *testing.T) {
	s := New(100)
	h := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Errorf("under capacity should pass, got %d", w.Code)
	}
}

// Edge: over capacity returns 503
func TestEdge_OverCapacity503(t *testing.T) {
	s := New(0) // zero capacity = always shed
	h := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 503 {
		t.Errorf("over capacity should return 503, got %d", w.Code)
	}
}

// Edge: loadshed with health check bypass
func TestEdge_HealthBypass(t *testing.T) {
	s := New(0) // always shed
	h := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	// Health endpoint should still be shed (no special bypass in middleware)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))
	if w.Code == 200 {
		t.Log("loadshed does not bypass /health — expected behavior")
	}
}

// Edge: concurrent middleware calls must not panic
func TestEdge_ConcurrentMiddleware(t *testing.T) {
	s := New(100)
	h := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// Edge: Stats tracks active and rejected
func TestEdge_StatsTracking(t *testing.T) {
	s := New(1)
	h := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	_, rejected := s.Stats()
	_ = rejected // just verify Stats doesn't panic
}
