package loadshed

import (
	"net/http"
	"net/http/httptest"
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
