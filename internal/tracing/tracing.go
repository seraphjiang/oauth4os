// Package tracing provides OpenTelemetry instrumentation for oauth4os proxy stages.
//
// Each proxy stage (JWT validation, scope mapping, Cedar evaluation, upstream)
// gets its own span under a parent request span.
//
// Enable by setting OTEL_EXPORTER_OTLP_ENDPOINT. Without it, uses a no-op tracer.
package tracing

import (
	"context"
	"net/http"
	"os"
	"time"
)

// Span represents a trace span for a proxy stage.
type Span struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Attrs     map[string]string
	Status    string // "ok", "error"
	Children  []*Span
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
