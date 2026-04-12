package timeout

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Property: instant handler never returns 504
func TestProperty_InstantNever504(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := Middleware(inner, 1*time.Second)
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 {
			t.Fatalf("instant handler should never 504, got %d on iteration %d", w.Code, i)
		}
	}
}

// Property: very slow handler always returns 504
func TestProperty_SlowAlways504(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // wait for cancellation
	})
	handler := Middleware(inner, 10*time.Millisecond)
	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 504 {
			t.Fatalf("slow handler should always 504, got %d on iteration %d", w.Code, i)
		}
	}
}
