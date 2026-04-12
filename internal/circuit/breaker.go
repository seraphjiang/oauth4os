// Package circuit implements a simple circuit breaker for upstream requests.
package circuit

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	Closed   State = iota // normal — requests pass through
	Open                  // tripped — requests rejected
	HalfOpen              // testing — one request allowed
)

// Breaker tracks upstream failures and opens the circuit after threshold consecutive 5xx.
type Breaker struct {
	mu        sync.Mutex
	state     State
	failures  int
	threshold int
	cooldown  time.Duration
	openedAt  time.Time
}

// New creates a breaker that opens after threshold consecutive 5xx errors,
// staying open for cooldown duration before allowing a probe request.
func New(threshold int, cooldown time.Duration) *Breaker {
	return &Breaker{threshold: threshold, cooldown: cooldown}
}

// Allow returns true if the request should proceed.
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case Closed:
		return true
	case Open:
		if time.Since(b.openedAt) >= b.cooldown {
			b.state = HalfOpen
			return true
		}
		return false
	case HalfOpen:
		return false // only one probe at a time
	}
	return true
}

// Record reports the HTTP status code of an upstream response.
func (b *Breaker) Record(statusCode int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if statusCode >= 500 {
		b.failures++
		if b.failures >= b.threshold {
			b.state = Open
			b.openedAt = time.Now()
		}
	} else {
		b.failures = 0
		b.state = Closed
	}
}

// State returns the current circuit breaker state.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

// RetryAfter returns seconds until the circuit may close, or 0 if closed.
func (b *Breaker) RetryAfter() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state != Open {
		return 0
	}
	remaining := b.cooldown - time.Since(b.openedAt)
	if remaining <= 0 {
		return 1
	}
	return int(remaining.Seconds()) + 1
}

// Middleware short-circuits requests when the breaker is open, returning 503 + Retry-After.
func (b *Breaker) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !b.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", fmt.Sprintf("%d", b.RetryAfter()))
			w.WriteHeader(http.StatusServiceUnavailable)
			reqID := w.Header().Get("X-Request-ID")
			fmt.Fprintf(w, `{"error":"circuit_open","request_id":%q,"retry_after":%d}`, reqID, b.RetryAfter())
			return
		}
		rec := &statusRecorder{ResponseWriter: w, code: 200}
		next.ServeHTTP(rec, r)
		b.Record(rec.code)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}
