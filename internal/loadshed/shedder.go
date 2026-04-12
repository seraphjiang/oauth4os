// Package loadshed rejects requests when concurrency exceeds a threshold.
package loadshed

import (
	"net/http"
	"sync/atomic"
)

// Shedder tracks in-flight requests and rejects when over capacity.
type Shedder struct {
	inflight atomic.Int64
	max      int64
	shed     atomic.Int64 // total shed count
}

// New creates a shedder with the given max concurrent requests.
func New(maxConcurrent int) *Shedder {
	return &Shedder{max: int64(maxConcurrent)}
}

// Middleware wraps an http.Handler with load shedding.
func (s *Shedder) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := s.inflight.Add(1)
		if cur > s.max {
			s.inflight.Add(-1)
			s.shed.Add(1)
			w.Header().Set("Retry-After", "5")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"load_shed","message":"server at capacity"}`))
			return
		}
		defer s.inflight.Add(-1)
		next.ServeHTTP(w, r)
	})
}

// Stats returns current inflight count and total shed count.
func (s *Shedder) Stats() (inflight, shed int64) {
	return s.inflight.Load(), s.shed.Load()
}
