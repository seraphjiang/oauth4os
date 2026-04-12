package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mutation: remove span from context → FromContext must return span set by middleware
func TestMutation_SpanInContext(t *testing.T) {
	var gotSpan *Span
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSpan = FromContext(r.Context())
		w.WriteHeader(200)
	})
	handler := Middleware(inner, &NoopTracer{})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	if gotSpan == nil {
		t.Error("middleware must inject span into context")
	}
}

// Mutation: remove FromContext nil safety → empty context must return nil
func TestMutation_EmptyContext(t *testing.T) {
	span := FromContext(context.Background())
	if span != nil {
		t.Error("empty context should return nil span")
	}
}

// Mutation: remove traceparent parsing → must handle W3C traceparent header
func TestMutation_TraceparentParsing(t *testing.T) {
	parts := splitTraceparent("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	if len(parts) != 4 {
		t.Errorf("expected 4 parts, got %d", len(parts))
	}
}

// Mutation: remove invalid traceparent handling → bad header must not panic
func TestMutation_BadTraceparent(t *testing.T) {
	parts := splitTraceparent("")
	if len(parts) == 4 {
		t.Error("empty traceparent should not parse as valid")
	}
	parts = splitTraceparent("not-valid")
	_ = parts // must not panic
}
