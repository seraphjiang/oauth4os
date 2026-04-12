package tracing

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseTraceparent(t *testing.T) {
	tid, pid := parseTraceparent("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	if tid != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("trace-id: got %s", tid)
	}
	if pid != "00f067aa0ba902b7" {
		t.Fatalf("parent-id: got %s", pid)
	}
}

func TestParseTraceparentInvalid(t *testing.T) {
	tid, _ := parseTraceparent("garbage")
	if tid != "" {
		t.Fatal("should return empty for invalid")
	}
}

func TestFormatTraceparent(t *testing.T) {
	tp := formatTraceparent("4bf92f3577b34da6a3ce929d0e0e4736", "00f067aa0ba902b7")
	if tp != "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
		t.Fatalf("got %s", tp)
	}
}

func TestInjectTraceparent(t *testing.T) {
	tracer := NewStdoutTracer(nil)
	ctx, span := tracer.StartSpan(nil, "test", nil)
	_ = span
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(ctx)
	InjectTraceparent(r)
	if r.Header.Get("traceparent") == "" {
		t.Fatal("expected traceparent header")
	}
}

func TestPropagateMiddleware(t *testing.T) {
	tracer := &CollectingTracer{}
	var gotTraceID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if span := FromContext(r.Context()); span != nil {
			gotTraceID = span.TraceID
		}
		w.WriteHeader(200)
	})

	handler := PropagateMiddleware(inner, tracer)
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if gotTraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("expected propagated trace-id, got %s", gotTraceID)
	}
	if w.Header().Get("traceparent") == "" {
		t.Fatal("expected traceparent in response")
	}
}
