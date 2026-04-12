// Package tracing provides OpenTelemetry-style instrumentation for oauth4os proxy stages.
package tracing

import (
	"context"
	"fmt"
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
// Supports W3C traceparent header for distributed trace correlation.
func Middleware(next http.Handler, tracer Tracer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse incoming W3C traceparent: 00-<trace_id>-<parent_id>-<flags>
		var parentCtx context.Context = r.Context()
		if tp := r.Header.Get("traceparent"); len(tp) >= 55 {
			parts := splitTraceparent(tp)
			if parts != nil {
				parentCtx = context.WithValue(parentCtx, spanKey{}, &Span{
					TraceID: parts[1],
					SpanID:  parts[2],
				})
			}
		}

		ctx, span := tracer.StartSpan(parentCtx, string(SpanRequest), map[string]string{
			"http.method": r.Method,
			"http.path":   r.URL.Path,
		})

		// Set W3C traceparent on response + upstream propagation
		tp := fmt.Sprintf("00-%s-%s-01", span.TraceID, span.SpanID)
		w.Header().Set("traceparent", tp)
		w.Header().Set("X-Trace-ID", span.TraceID)
		r.Header.Set("traceparent", tp)

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
		tracer.EndSpan(span, "ok")
	})
}

// splitTraceparent parses "00-traceid-parentid-flags" into parts.
func splitTraceparent(tp string) []string {
	parts := make([]string, 0, 4)
	start := 0
	for i := 0; i <= len(tp); i++ {
		if i == len(tp) || tp[i] == '-' {
			parts = append(parts, tp[start:i])
			start = i + 1
		}
	}
	if len(parts) != 4 {
		return nil
	}
	return parts
}
