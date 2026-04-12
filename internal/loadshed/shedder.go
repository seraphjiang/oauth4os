// Package loadshed rejects requests when the proxy is overloaded.
package loadshed

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// Shedder tracks active requests and rejects new ones above the threshold.
type Shedder struct {
	active    atomic.Int64
	threshold int64
	rejected  atomic.Int64
}

// New creates a load shedder with the given max concurrent request threshold.
func New(maxConcurrent int) *Shedder {
	return &Shedder{threshold: int64(maxConcurrent)}
}

// Middleware rejects requests with 503 when active requests exceed threshold.
func (s *Shedder) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := s.active.Add(1)
		defer s.active.Add(-1)
		if current > s.threshold {
			s.rejected.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "5")
			w.WriteHeader(http.StatusServiceUnavailable)
			reqID := w.Header().Get("X-Request-ID")
			fmt.Fprintf(w, `{"error":"overloaded","request_id":%q,"active":%d,"threshold":%d}`, reqID, current-1, s.threshold)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Stats returns current active requests and total rejected count.
func (s *Shedder) Stats() (active, rejected int64) {
	return s.active.Load(), s.rejected.Load()
}
