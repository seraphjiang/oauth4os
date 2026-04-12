// Package tracing provides span-based tracing for oauth4os proxy stages.
package tracing

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

// Span represents a trace span.
type Span struct {
	TraceID   string            `json:"trace_id,omitempty"`
	SpanID    string            `json:"span_id,omitempty"`
	ParentID  string            `json:"parent_id,omitempty"`
	Name      string            `json:"name"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time"`
	Duration  time.Duration     `json:"duration"`
	Attrs     map[string]string `json:"attrs,omitempty"`
	Status    string            `json:"status"`
	Children  []*Span           `json:"children,omitempty"`
}

type spanKey struct{}

// FromContext retrieves the current span from context.
func FromContext(ctx context.Context) *Span {
	if s, ok := ctx.Value(spanKey{}).(*Span); ok {
		return s
	}
	return nil
}

// Tracer is the interface for span-based tracing.
type Tracer interface {
	StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span)
	EndSpan(span *Span, status string)
}

// NoopTracer does nothing.
type NoopTracer struct{}

func (NoopTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span) {
	s := &Span{Name: name, StartTime: time.Now(), Attrs: attrs}
	return context.WithValue(ctx, spanKey{}, s), s
}

func (NoopTracer) EndSpan(s *Span, status string) {
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)
	s.Status = status
}

// StdoutTracer exports spans as JSON lines.
type StdoutTracer struct {
	enc *json.Encoder
	mu  sync.Mutex
}

// NewStdoutTracer creates a tracer that writes JSON spans to w.
func NewStdoutTracer(w io.Writer) *StdoutTracer {
	return &StdoutTracer{enc: json.NewEncoder(w)}
}

func (t *StdoutTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span) {
	s := &Span{Name: name, StartTime: time.Now(), Attrs: attrs}
	if parent := FromContext(ctx); parent != nil {
		s.ParentID = parent.SpanID
		s.TraceID = parent.TraceID
	}
	return context.WithValue(ctx, spanKey{}, s), s
}

func (t *StdoutTracer) EndSpan(s *Span, status string) {
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)
	s.Status = status
	t.mu.Lock()
	t.enc.Encode(s)
	t.mu.Unlock()
}

// Middleware wraps an http.Handler with request-level tracing.
func Middleware(next http.Handler, tracer Tracer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.StartSpan(r.Context(), "request", map[string]string{
			"http.method": r.Method,
			"http.path":   r.URL.Path,
		})
		r = r.WithContext(ctx)
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		status := "ok"
		if sw.status >= 400 {
			status = "error"
		}
		tracer.EndSpan(span, status)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// TracerIface is an alias for backward compatibility.
type TracerIface = Tracer

// Span kind constants for proxy stages.
const (
	SpanRequest  = "request"
	SpanJWT      = "jwt.validate"
	SpanScope    = "scope.map"
	SpanCedar    = "cedar.evaluate"
	SpanUpstream = "upstream.forward"
)
