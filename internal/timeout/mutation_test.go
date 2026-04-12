package timeout

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Mutation: remove context cancellation → context must be cancelled after timeout
func TestMutation_ContextCancelled(t *testing.T) {
	var ctxErr error
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		ctxErr = r.Context().Err()
	})
	handler := Middleware(inner, 20*time.Millisecond)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if ctxErr == nil {
		t.Error("context must be cancelled on timeout")
	}
}

// Mutation: remove 504 status → timed out requests must get 504
func TestMutation_504Status(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	handler := Middleware(inner, 20*time.Millisecond)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 504 {
		t.Errorf("timeout must return 504, got %d", w.Code)
	}
}

// Mutation: remove JSON body → 504 response must contain error details
func TestMutation_504Body(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	handler := Middleware(inner, 20*time.Millisecond)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Body.Len() == 0 {
		t.Error("504 response must include error body")
	}
}

// Mutation: pass through must preserve original status
func TestMutation_PreserveStatus(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	handler := Middleware(inner, 1*time.Second)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 201 {
		t.Errorf("should preserve 201, got %d", w.Code)
	}
}
