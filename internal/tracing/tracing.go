// Package tracing provides OpenTelemetry-style instrumentation for oauth4os proxy stages.
package tracing

import (
	"context"
	"net/http"
	"time"
)

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
}

type spanKey struct{}

// FromContext retrieves the current span from context.
func FromContext(ctx context.Context) *Span {
	if s, ok := ctx.Value(spanKey{}).(*Span); ok {
		return s
	}
	return nil
}

// Tracer creates and manages spans.
type Tracer interface {
	StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span)
	EndSpan(span *Span, status string)
}

// TracerIface is an alias for backward compatibility.
type TracerIface = Tracer

// Middleware wraps an http.Handler with request-level tracing.
func Middleware(next http.Handler, tracer Tracer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.StartSpan(r.Context(), string(SpanRequest), map[string]string{
			"http.method": r.Method,
			"http.path":   r.URL.Path,
		})
		w.Header().Set("X-Trace-ID", span.TraceID)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
		tracer.EndSpan(span, "ok")
	})
}
