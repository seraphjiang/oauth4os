package circuit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClosedAllows(t *testing.T) {
	b := New(3, 1*time.Second)
	if !b.Allow() {
		t.Error("closed breaker should allow")
	}
}

func TestOpensAfterThreshold(t *testing.T) {
	b := New(3, 1*time.Second)
	for i := 0; i < 3; i++ {
		b.Record(500)
	}
	if b.Allow() {
		t.Error("breaker should be open after 3 failures")
	}
}

func TestSuccessResets(t *testing.T) {
	b := New(3, 1*time.Second)
	b.Record(500)
	b.Record(500)
	b.Record(200) // resets
	b.Record(500)
	b.Record(500)
	if !b.Allow() {
		t.Error("success should reset failure count")
	}
}

func TestHalfOpenAfterCooldown(t *testing.T) {
	b := New(1, 50*time.Millisecond)
	b.Record(500)
	if b.Allow() {
		t.Error("should be open")
	}
	time.Sleep(100 * time.Millisecond)
	if !b.Allow() {
		t.Error("should transition to half-open after cooldown")
	}
}

func TestMiddleware503(t *testing.T) {
	b := New(1, 1*time.Second)
	b.Record(500) // open

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach inner handler")
	})
	handler := b.Middleware(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	if w.Code != 503 {
		t.Errorf("expected 503, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("missing Retry-After header")
	}
}

func TestRetryAfter(t *testing.T) {
	b := New(1, 5*time.Second)
	if b.RetryAfter() != 0 {
		t.Error("closed breaker should return 0")
	}
	b.Record(500)
	ra := b.RetryAfter()
	if ra < 1 || ra > 6 {
		t.Errorf("expected retry-after 1-6, got %d", ra)
	}
}
