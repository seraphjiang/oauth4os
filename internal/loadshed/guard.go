// Package loadshed rejects requests when concurrent queue depth exceeds a threshold.
package loadshed

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// Guard tracks in-flight requests and rejects when over threshold.
type Guard struct {
	inflight  atomic.Int64
	threshold int64
	rejected  atomic.Int64
}

// New creates a load shedding guard. threshold=0 disables.
func New(threshold int) *Guard {
	return &Guard{threshold: int64(threshold)}
}

// Rejected returns total rejected count.
func (g *Guard) Rejected() int64 { return g.rejected.Load() }

// Inflight returns current in-flight count.
func (g *Guard) Inflight() int64 { return g.inflight.Load() }

// Middleware wraps a handler with load shedding.
func (g *Guard) Middleware(next http.Handler) http.Handler {
	if g.threshold <= 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := g.inflight.Add(1)
		defer g.inflight.Add(-1)
		if cur > g.threshold {
			g.rejected.Add(1)
			w.Header().Set("Retry-After", "5")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"error":"server_busy","message":"load shedding active"}`)
			return
		}
		next.ServeHTTP(w, r)
	})
}
