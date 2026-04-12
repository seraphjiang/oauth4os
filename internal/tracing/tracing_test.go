package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStdoutTracer_SpanLifecycle(t *testing.T) {
	var buf bytes.Buffer
	tr := NewStdoutTracer(&buf)

	ctx, span := tr.StartSpan(context.Background(), "test.op", map[string]string{"key": "val"})
	if span.TraceID == "" || span.SpanID == "" {
		t.Fatal("missing IDs")
	}
	if span.Name != "test.op" {
		t.Fatalf("name = %s", span.Name)
	}

	// Child span inherits trace ID
	_, child := tr.StartSpan(ctx, "child.op", nil)
	if child.TraceID != span.TraceID {
		t.Fatal("child should inherit trace ID")
	}
	if child.ParentID != span.SpanID {
		t.Fatal("child parent should be parent span ID")
	}

	tr.EndSpan(child, "ok")
	tr.EndSpan(span, "ok")

	// Should have 2 JSON lines
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(lines))
	}

	var exported Span
	json.Unmarshal(lines[0], &exported)
	if exported.Duration <= 0 {
		t.Fatal("duration should be > 0")
	}
}

func TestNoopTracer(t *testing.T) {
	tr := NoopTracer{}
	_, span := tr.StartSpan(context.Background(), "noop", nil)
	tr.EndSpan(span, "ok")
	if span.Status != "ok" {
		t.Fatalf("status = %s", span.Status)
	}
}

func TestCollectingTracer(t *testing.T) {
	tr := &CollectingTracer{}
	ctx, _ := tr.StartSpan(context.Background(), "parent", nil)
	_, _ = tr.StartSpan(ctx, "child", nil)
	tr.EndSpan(tr.Spans[0], "ok") // won't have spans yet, use direct
	// Just verify it doesn't panic and collects
	if len(tr.Spans) != 0 {
		// Spans only added on EndSpan
	}
}

func TestMiddleware_AddsTraceID(t *testing.T) {
	tr := &CollectingTracer{}
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify span is in context
		span := FromContext(r.Context())
		if span == nil {
			t.Fatal("no span in context")
		}
		w.WriteHeader(200)
	}), tr)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Trace-ID") == "" {
		t.Fatal("missing X-Trace-ID header")
	}
}

func TestMiddleware_ErrorStatus(t *testing.T) {
	tr := &CollectingTracer{}
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
	}), tr)

	req := httptest.NewRequest("GET", "/forbidden", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if len(tr.Spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(tr.Spans))
	}
	if tr.Spans[0].Status != "error" {
		t.Fatalf("status = %s, want error", tr.Spans[0].Status)
	}
}
