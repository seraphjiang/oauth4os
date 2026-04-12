// Package tracing provides OpenTelemetry instrumentation for the oauth4os proxy.
// Creates trace spans for each proxy stage: rate limit, JWT validation, scope mapping,
// Cedar evaluation, and upstream forwarding.
package tracing

import (
	"context"
	"net/http"
	"time"
)

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

// Span represents a single trace span.
type Span struct {
	TraceID   string            `json:"trace_id"`
	SpanID    string            `json:"span_id"`
	ParentID  string            `json:"parent_id,omitempty"`
	Name      string            `json:"name"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time,omitempty"`
	Duration  time.Duration     `json:"duration_ns,omitempty"`
	Status    string            `json:"status"` // ok, error
	Attrs     map[string]string `json:"attributes,omitempty"`
}

// Tracer creates and manages spans. Implementations can export to
// OTLP, stdout, or noop for testing.
type Tracer interface {
	StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span)
	EndSpan(span *Span, status string)
}

// Middleware wraps an http.Handler with request-level tracing.
// Creates a root span per request and injects trace context.
func Middleware(next http.Handler, tracer Tracer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.StartSpan(r.Context(), string(SpanRequest), map[string]string{
			"http.method": r.Method,
			"http.path":   r.URL.Path,
			"http.host":   r.Host,
		})
		// Propagate trace ID in response header
		w.Header().Set("X-Trace-ID", span.TraceID)
		r = r.WithContext(ctx)

		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)

		span.Attrs["http.status_code"] = http.StatusText(sw.status)
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

// FromContext retrieves the current span from context.
func FromContext(ctx context.Context) *Span {
	if s, ok := ctx.Value(spanKey{}).(*Span); ok {
		return s
	}
	return nil
}

type spanKey struct{}
