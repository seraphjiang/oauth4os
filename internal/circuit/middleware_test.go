package circuit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Test middleware passes through when closed
func TestMiddleware_PassThrough(t *testing.T) {
	b := New(10, 1*time.Second)
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	})
	handler := b.Middleware(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	if !called {
		t.Error("inner handler should be called when circuit closed")
	}
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// Test middleware records upstream status
func TestMiddleware_RecordsStatus(t *testing.T) {
	b := New(2, 1*time.Second)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
	})
	handler := b.Middleware(inner)

	// Two 502s should trip the breaker
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	}

	// Next request should get 503 from breaker
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	if w.Code != 503 {
		t.Errorf("expected 503 from open breaker, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "circuit_open") {
		t.Error("503 body should contain circuit_open")
	}
}
