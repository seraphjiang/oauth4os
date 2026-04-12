// Package chaos provides middleware for fault injection testing.
package chaos

import (
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Config controls fault injection behavior.
type Config struct {
	ErrorRate    float64       // probability of returning 500 (0.0–1.0)
	LatencyMin   time.Duration // minimum injected latency
	LatencyMax   time.Duration // maximum injected latency
	LatencyRate  float64       // probability of injecting latency (0.0–1.0)
	DropRate     float64       // probability of dropping connection
}

// Injector injects faults into HTTP requests.
type Injector struct {
	cfg     atomic.Value // *Config
	enabled atomic.Bool
	mu      sync.Mutex
	rng     *rand.Rand
}

func (inj *Injector) randFloat() float64 {
	inj.mu.Lock()
	v := inj.rng.Float64()
	inj.mu.Unlock()
	return v
}

func (inj *Injector) randDuration(min, max time.Duration) time.Duration {
	inj.mu.Lock()
	d := min + time.Duration(inj.rng.Int63n(int64(max-min)))
	inj.mu.Unlock()
	return d
}

// New creates a disabled fault injector.
func New(cfg Config) *Injector {
	inj := &Injector{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
	inj.cfg.Store(&cfg)
	return inj
}

// Enable turns on fault injection.
func (inj *Injector) Enable()  { inj.enabled.Store(true) }

// Disable turns off fault injection.
func (inj *Injector) Disable() { inj.enabled.Store(false) }

// SetConfig updates the fault injection config.
func (inj *Injector) SetConfig(cfg Config) { inj.cfg.Store(&cfg) }

// Config returns the current config.
func (inj *Injector) GetConfig() Config { return *inj.cfg.Load().(*Config) }

// Stats tracks injected faults.
type Stats struct {
	Errors  atomic.Int64
	Delays  atomic.Int64
	Drops   atomic.Int64
	Total   atomic.Int64
}

// Middleware wraps an HTTP handler with fault injection.
func (inj *Injector) Middleware(next http.Handler) http.Handler {
	var stats Stats
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stats.Total.Add(1)
		if !inj.enabled.Load() {
			next.ServeHTTP(w, r)
			return
		}
		cfg := inj.cfg.Load().(*Config)

		// Connection drop
		if cfg.DropRate > 0 && inj.randFloat() < cfg.DropRate {
			stats.Drops.Add(1)
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, err := hj.Hijack()
				if err == nil {
					conn.Close()
					return
				}
			}
			// Fallback: just close without response
			return
		}

		// Latency injection
		if cfg.LatencyRate > 0 && inj.randFloat() < cfg.LatencyRate {
			stats.Delays.Add(1)
			delay := cfg.LatencyMin
			if cfg.LatencyMax > cfg.LatencyMin {
				delay = inj.randDuration(cfg.LatencyMin, cfg.LatencyMax)
			}
			time.Sleep(delay)
		}

		// Error injection
		if cfg.ErrorRate > 0 && inj.randFloat() < cfg.ErrorRate {
			stats.Errors.Add(1)
			http.Error(w, `{"error":"chaos_fault","message":"injected server error"}`, http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}
