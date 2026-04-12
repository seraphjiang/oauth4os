// Package accesslog provides JSON-structured HTTP access logging middleware.
package accesslog

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// Entry is one JSON access log line.
type Entry struct {
	Timestamp string `json:"ts"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
	ClientID  string `json:"client_id,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Size      int    `json:"size"`
	UserAgent string `json:"user_agent,omitempty"`
}

// Logger writes JSON access logs.
type Logger struct {
	out io.Writer
	enc *json.Encoder
}

// New creates an access logger writing to out.
func New(out io.Writer) *Logger {
	return &Logger{out: out, enc: json.NewEncoder(out)}
}

type responseCapture struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *responseCapture) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseCapture) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.size += n
	return n, err
}

// Middleware wraps a handler with access logging. getClientID extracts client from request (can be nil).
func (l *Logger) Middleware(next http.Handler, getClientID func(r *http.Request) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rc := &responseCapture{ResponseWriter: w, status: 200}
		next.ServeHTTP(rc, r)

		cid := ""
		if getClientID != nil {
			cid = getClientID(r)
		}
		l.enc.Encode(Entry{
			Timestamp: start.UTC().Format(time.RFC3339),
			Method:    r.Method,
			Path:      r.URL.Path,
			Status:    rc.status,
			LatencyMs: time.Since(start).Milliseconds(),
			ClientID:  cid,
			RequestID: w.Header().Get("X-Request-ID"),
			Size:      rc.size,
			UserAgent: r.UserAgent(),
		})
	})
}
