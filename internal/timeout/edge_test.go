package timeout

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Edge: fast handler completes normally
func TestEdge_FastHandlerCompletes(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}), time.Second)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Errorf("fast handler should return 200, got %d", w.Code)
	}
}

// Edge: slow handler gets 504
func TestEdge_SlowHandler504(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}), 50*time.Millisecond)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 504 {
		t.Errorf("slow handler should return 504, got %d", w.Code)
	}
}
