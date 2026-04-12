// Package tracing provides OpenTelemetry instrumentation for oauth4os proxy stages.
//
// Each proxy stage (JWT validation, scope mapping, Cedar evaluation, upstream)
// gets its own span under a parent request span.
//
// Enable by setting OTEL_EXPORTER_OTLP_ENDPOINT. Without it, uses a no-op tracer.
package tracing

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// Span represents a trace span for a proxy stage.
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

// FromContext retrieves the current span from context.
func FromContext(ctx context.Context) *Span {
	if s, ok := ctx.Value(spanKey{}).(*Span); ok {
		return s
	}
	return nil
}

// Tracer creates spans for proxy stages.
type Tracer struct {
	enabled    bool
	exportFunc func(*Span)
}

// New creates a tracer. Enabled if OTEL_EXPORTER_OTLP_ENDPOINT is set.
func New(exportFn func(*Span)) *Tracer {
	return &Tracer{
		enabled:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "",
		exportFunc: exportFn,
	}
}

type spanKey struct{}

// StartRequest begins a root span for an HTTP request.
func (t *Tracer) StartRequest(ctx context.Context, r *http.Request) (context.Context, *Span) {
	s := &Span{
		Name:      "proxy.request",
		StartTime: time.Now(),
		Attrs: map[string]string{
			"http.method": r.Method,
			"http.path":   r.URL.Path,
			"http.host":   r.Host,
		},
	}
	return context.WithValue(ctx, spanKey{}, s), s
}

// StartStage begins a child span for a proxy stage (jwt, scope, cedar, upstream).
func (t *Tracer) StartStage(ctx context.Context, name string) *Span {
	child := &Span{
		Name:      "proxy." + name,
		StartTime: time.Now(),
		Attrs:     map[string]string{},
	}
	if parent, ok := ctx.Value(spanKey{}).(*Span); ok {
		parent.Children = append(parent.Children, child)
	}
	return child
}

// End finishes a span.
func (t *Tracer) End(s *Span, err error) {
	s.EndTime = time.Now()
	if err != nil {
		s.Status = "error"
		s.Attrs["error"] = err.Error()
	} else {
		s.Status = "ok"
	}
}

// FinishRequest ends the root span and exports if enabled.
func (t *Tracer) FinishRequest(s *Span, statusCode int) {
	s.EndTime = time.Now()
	s.Attrs["http.status_code"] = http.StatusText(statusCode)
	if statusCode >= 400 {
		s.Status = "error"
	} else {
		s.Status = "ok"
	}
	if t.enabled && t.exportFunc != nil {
		t.exportFunc(s)
	}
}

// ── Interface-based tracing (used by proxy middleware) ─────────────────────────

// SpanKind identifies the type of span.
type SpanKind string

const (
	SpanRequest   SpanKind = "request"
	SpanRateLimit SpanKind = "ratelimit"
	SpanJWT       SpanKind = "jwt.validate"
	SpanScope     SpanKind = "scope.map"
	SpanCedar     SpanKind = "cedar.evaluate"
	SpanUpstream  SpanKind = "upstream.forward"
)

// TracerI is the interface for span-based tracing.
type TracerI interface {
	StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span)
	EndSpan(span *Span, status string)
}

// Alias so main.go can use tracing.Tracer as the interface.
type Tracer = TracerI

// StdoutTracer exports spans as JSON lines.
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
	s := &Span{Name: name, StartTime: time.Now(), Attrs: attrs, Status: "ok"}
	return context.WithValue(ctx, spanKey{}, s), s
}

func (t *StdoutTracer) EndSpan(s *Span, status string) {
	s.EndTime = time.Now()
	s.Status = status
	t.mu.Lock()
	t.enc.Encode(s)
	t.mu.Unlock()
}

// NoopTracer does nothing.
type NoopTracer struct{}

func (NoopTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span) {
	s := &Span{Name: name, StartTime: time.Now(), Attrs: attrs}
	return context.WithValue(ctx, spanKey{}, s), s
}

func (NoopTracer) EndSpan(s *Span, status string) {
	s.EndTime = time.Now()
	s.Status = status
}

// Middleware wraps an http.Handler with request-level tracing.
func Middleware(next http.Handler, tracer TracerI) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.StartSpan(r.Context(), string(SpanRequest), map[string]string{
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

// SpanKind constants for proxy stages.
type SpanKind string

const (
	SpanRequest   SpanKind = "request"
	SpanRateLimit SpanKind = "ratelimit"
	SpanJWT       SpanKind = "jwt.validate"
	SpanScope     SpanKind = "scope.map"
	SpanCedar     SpanKind = "cedar.evaluate"
	SpanUpstream  SpanKind = "upstream.forward"
)

// --- Compatibility interface used by cmd/proxy/main.go ---

// TracerIface is the interface main.go expects.
type TracerIface interface {
	StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span)
	EndSpan(span *Span, status string)
}

// NoopTracer does nothing.
type NoopTracer struct{}

func (NoopTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span) {
	s := &Span{Name: name, StartTime: time.Now(), Attrs: attrs, Status: "ok"}
	return context.WithValue(ctx, spanKey{}, s), s
}
func (NoopTracer) EndSpan(s *Span, status string) {
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)
	s.Status = status
}

// StdoutTracer logs spans as JSON to stderr.
type StdoutTracer struct{ w *os.File }

func NewStdoutTracer(w *os.File) *StdoutTracer { return &StdoutTracer{w: w} }

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
}

// Middleware wraps an http.Handler with request-level tracing.
func Middleware(next http.Handler, tracer TracerIface) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.StartSpan(r.Context(), string(SpanRequest), map[string]string{
			"http.method": r.Method,
			"http.path":   r.URL.Path,
		})
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
		tracer.EndSpan(span, "ok")
	})
}
