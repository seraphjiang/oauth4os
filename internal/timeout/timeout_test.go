package timeout

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFastHandler(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	handler := Middleware(inner, 1*time.Second)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSlowHandler(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(5 * time.Second):
			w.WriteHeader(200)
		}
	})
	handler := Middleware(inner, 50*time.Millisecond)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/slow", nil))
	if w.Code != 504 {
		t.Errorf("expected 504 for slow handler, got %d", w.Code)
	}
}
