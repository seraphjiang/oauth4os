package timeout

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFastHandler(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	})
	h := Middleware(inner, 1*time.Second)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestSlowHandler(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(5 * time.Second):
		case <-r.Context().Done():
		}
	})
	h := Middleware(inner, 50*time.Millisecond)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 504 {
		t.Fatalf("expected 504, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "request_timeout") {
		t.Fatal("expected timeout error in body")
	}
}

func TestTimeoutPreservesHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(201)
	})
	h := Middleware(inner, 1*time.Second)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	if w.Header().Get("X-Custom") != "value" {
		t.Fatal("custom header not preserved")
	}
}
