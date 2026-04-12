package timeout

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Property: handler always completes (no goroutine leak) regardless of timeout
func TestProperty_AlwaysCompletes(t *testing.T) {
	for i := 0; i < 50; i++ {
		d := time.Duration(rand.Intn(100)+1) * time.Millisecond
		sleep := time.Duration(rand.Intn(50)) * time.Millisecond
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(sleep):
				w.WriteHeader(200)
			}
		})
		handler := Middleware(inner, d)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 && w.Code != 504 {
			t.Errorf("unexpected status %d (timeout=%v sleep=%v)", w.Code, d, sleep)
		}
	}
}

// Property: 504 only when handler is slower than timeout
func TestProperty_504OnlyWhenSlow(t *testing.T) {
	for i := 0; i < 20; i++ {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200) // instant
		})
		handler := Middleware(inner, 1*time.Second)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 {
			t.Errorf("instant handler should never 504, got %d", w.Code)
		}
	}
}
