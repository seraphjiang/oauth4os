package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"sync"
	"time"
)

// StdoutTracer exports spans as JSON lines to a writer. Good for dev/debug.
type StdoutTracer struct {
	w   io.Writer
	enc *json.Encoder
	mu  sync.Mutex
}

func NewStdoutTracer(w io.Writer) *StdoutTracer {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &StdoutTracer{w: w, enc: enc}
}

func (t *StdoutTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span) {
	span := &Span{
		TraceID:   traceIDFromContext(ctx),
		SpanID:    genID(8),
		Name:      name,
		StartTime: time.Now(),
		Status:    "ok",
		Attrs:     attrs,
	}
	if parent := FromContext(ctx); parent != nil {
		span.ParentID = parent.SpanID
		span.TraceID = parent.TraceID
	}
	if span.TraceID == "" {
		span.TraceID = genID(16)
	}
	return context.WithValue(ctx, spanKey{}, span), span
}

func (t *StdoutTracer) EndSpan(span *Span, status string) {
	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)
	span.Status = status
	t.mu.Lock()
	t.enc.Encode(span)
	t.mu.Unlock()
}

// NoopTracer does nothing. Use in tests or when tracing is disabled.
type NoopTracer struct{}

func (NoopTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span) {
	span := &Span{TraceID: "noop", SpanID: "noop", Name: name, StartTime: time.Now(), Attrs: attrs}
	return context.WithValue(ctx, spanKey{}, span), span
}

func (NoopTracer) EndSpan(span *Span, status string) {
	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)
	span.Status = status
}

// CollectingTracer collects spans in memory. Use in tests.
type CollectingTracer struct {
	Spans []*Span
	mu    sync.Mutex
}

func (t *CollectingTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span) {
	span := &Span{
		TraceID:   genID(16),
		SpanID:    genID(8),
		Name:      name,
		StartTime: time.Now(),
		Attrs:     attrs,
	}
	if parent := FromContext(ctx); parent != nil {
		span.ParentID = parent.SpanID
		span.TraceID = parent.TraceID
	}
	return context.WithValue(ctx, spanKey{}, span), span
}

func (t *CollectingTracer) EndSpan(span *Span, status string) {
	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)
	span.Status = status
	t.mu.Lock()
	t.Spans = append(t.Spans, span)
	t.mu.Unlock()
}

func genID(bytes int) string {
	b := make([]byte, bytes)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func traceIDFromContext(ctx context.Context) string {
	if s := FromContext(ctx); s != nil {
		return s.TraceID
	}
	return ""
}

// GenID generates a random hex ID (32 chars / 16 bytes).
func GenID() string { return genID(16) }
