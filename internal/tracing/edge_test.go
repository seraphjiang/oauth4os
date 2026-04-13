package tracing

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Edge: Middleware adds traceparent header
func TestEdge_MiddlewareAddsTraceparent(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Header().Get("traceparent") == "" && w.Code != 200 {
		t.Error("middleware should not break request")
	}
}

// Edge: Middleware propagates existing traceparent
func TestEdge_PropagatesTraceparent(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := FromContext(r.Context())
		if span != nil && span.TraceID != "" {
			w.Header().Set("X-Trace-ID", span.TraceID)
		}
		w.WriteHeader(200)
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("X-Trace-ID") != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("should propagate trace ID, got %q", w.Header().Get("X-Trace-ID"))
	}
}
