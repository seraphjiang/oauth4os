package tracing

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddleware_GeneratesTraceparent(t *testing.T) {
	tracer := &CollectingTracer{}
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), tracer)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))

	tp := rec.Header().Get("traceparent")
	if tp == "" {
		t.Fatal("expected traceparent header on response")
	}
	parts := strings.Split(tp, "-")
	if len(parts) != 4 || parts[0] != "00" || parts[3] != "01" {
		t.Fatalf("invalid traceparent format: %q", tp)
	}
	if len(parts[1]) != 32 {
		t.Fatalf("trace_id should be 32 hex chars, got %d", len(parts[1]))
	}
}

func TestMiddleware_PropagatesIncomingTrace(t *testing.T) {
	tracer := &CollectingTracer{}
	var capturedTP string
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTP = r.Header.Get("traceparent")
		w.WriteHeader(200)
	}), tracer)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("traceparent", "00-aaaabbbbccccddddeeee111122223333-1234567890abcdef-01")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Response should have same trace_id but new span_id
	tp := rec.Header().Get("traceparent")
	parts := strings.Split(tp, "-")
	if parts[1] != "aaaabbbbccccddddeeee111122223333" {
		t.Fatalf("trace_id should be propagated, got %q", parts[1])
	}
	if parts[2] == "1234567890abcdef" {
		t.Fatal("span_id should be new, not the parent's")
	}

	// Upstream request should also get the new traceparent
	if capturedTP == "" {
		t.Fatal("upstream request should have traceparent")
	}
}

func TestMiddleware_SetsXTraceID(t *testing.T) {
	tracer := &CollectingTracer{}
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), tracer)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Header().Get("X-Trace-ID") == "" {
		t.Fatal("expected X-Trace-ID header")
	}
}
