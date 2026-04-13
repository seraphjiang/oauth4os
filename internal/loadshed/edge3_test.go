package loadshed

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEdge_HighCapacity(t *testing.T) {
	s := New(10000)
	h := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 {
			t.Errorf("high capacity should pass all, got %d at %d", w.Code, i)
			break
		}
	}
}

func TestEdge_OneCapacity(t *testing.T) {
	s := New(1)
	h := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Errorf("first request should pass, got %d", w.Code)
	}
}
