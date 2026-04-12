// Package timeout provides per-request timeout middleware.
package timeout

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// Middleware wraps a handler with a request-scoped context deadline.
// If the handler doesn't complete within the timeout, the client gets 504.
func Middleware(next http.Handler, d time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		r = r.WithContext(ctx)

		done := make(chan struct{})
		tw := &timeoutWriter{ResponseWriter: w, code: 200}
		go func() {
			next.ServeHTTP(tw, r)
			close(done)
		}()

		select {
		case <-done:
			// Handler completed in time
		case <-ctx.Done():
			if !tw.written.Load() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusGatewayTimeout)
				reqID := w.Header().Get("X-Request-ID")
				fmt.Fprintf(w, `{"error":"request_timeout","request_id":%q,"timeout_ms":%d}`, reqID, d.Milliseconds())
			}
		}
	})
}

type timeoutWriter struct {
	http.ResponseWriter
	code    int
	written atomic.Bool
}

func (w *timeoutWriter) WriteHeader(code int) {
	if w.written.CompareAndSwap(false, true) {
		w.code = code
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *timeoutWriter) Write(b []byte) (int, error) {
	w.written.Store(true)
	return w.ResponseWriter.Write(b)
}
