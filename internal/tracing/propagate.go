package tracing

import (
	"fmt"
	"net/http"
)

// PropagateMiddleware extracts W3C traceparent from incoming requests and
// injects it into outgoing responses + upstream requests.
// Format: traceparent: 00-<trace-id>-<span-id>-<flags>
func PropagateMiddleware(next http.Handler, tracer Tracer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract incoming traceparent
		if tp := r.Header.Get("traceparent"); tp != "" {
			traceID, parentID := parseTraceparent(tp)
			if traceID != "" {
				ctx, span := tracer.StartSpan(r.Context(), "request", map[string]string{
					"http.method":    r.Method,
					"http.path":      r.URL.Path,
					"trace.parent":   parentID,
				})
				span.TraceID = traceID
				span.ParentID = parentID
				r = r.WithContext(ctx)
				w.Header().Set("traceparent", formatTraceparent(span.TraceID, span.SpanID))
				w.Header().Set("X-Trace-ID", span.TraceID)
				next.ServeHTTP(w, r)
				tracer.EndSpan(span, "ok")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// InjectTraceparent adds traceparent header to an outgoing request from context.
func InjectTraceparent(r *http.Request) {
	if span := FromContext(r.Context()); span != nil && span.TraceID != "" {
		r.Header.Set("traceparent", formatTraceparent(span.TraceID, span.SpanID))
	}
}

func formatTraceparent(traceID, spanID string) string {
	// Pad/truncate to spec: trace-id=32 hex, span-id=16 hex
	tid := padHex(traceID, 32)
	sid := padHex(spanID, 16)
	return fmt.Sprintf("00-%s-%s-01", tid, sid)
}

func parseTraceparent(tp string) (traceID, parentID string) {
	// Format: version-traceid-parentid-flags
	// 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
	if len(tp) < 55 {
		return "", ""
	}
	if tp[2] != '-' || tp[35] != '-' || tp[52] != '-' {
		return "", ""
	}
	return tp[3:35], tp[36:52]
}

func padHex(s string, n int) string {
	for len(s) < n {
		s = "0" + s
	}
	if len(s) > n {
		s = s[:n]
	}
	return s
}
