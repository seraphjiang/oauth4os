// Package histogram provides a lock-free request latency histogram for Prometheus exposition.
package histogram

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// Histogram tracks request latency distribution with fixed buckets.
type Histogram struct {
	buckets []float64 // upper bounds in seconds
	counts  []atomic.Int64
	sum     atomic.Int64 // microseconds
	count   atomic.Int64
	mu      sync.RWMutex
	byPath  map[string]*Histogram // per-endpoint breakdown
	depth   int
}

// New creates a histogram with standard latency buckets (in seconds).
func New() *Histogram {
	buckets := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	return &Histogram{
		buckets: buckets,
		counts:  make([]atomic.Int64, len(buckets)+1), // +1 for +Inf
		byPath:  make(map[string]*Histogram),
	}
}

// Observe records a latency observation.
func (h *Histogram) Observe(d time.Duration, path string) {
	secs := d.Seconds()
	h.sum.Add(int64(d.Microseconds()))
	h.count.Add(1)
	for i, b := range h.buckets {
		if secs <= b {
			h.counts[i].Add(1)
			break
		}
		if i == len(h.buckets)-1 {
			h.counts[len(h.buckets)].Add(1) // +Inf
		}
	}

	// Per-path tracking (1 level deep only)
	if h.depth == 0 && path != "" {
		h.mu.RLock()
		ph, ok := h.byPath[path]
		h.mu.RUnlock()
		if !ok {
			h.mu.Lock()
			ph, ok = h.byPath[path]
			if !ok {
				ph = &Histogram{
					buckets: h.buckets,
					counts:  make([]atomic.Int64, len(h.buckets)+1),
					depth:   1,
				}
				h.byPath[path] = ph
			}
			h.mu.Unlock()
		}
		ph.Observe(d, "")
	}
}

// WritePrometheus writes the histogram in Prometheus exposition format.
func (h *Histogram) WritePrometheus(w io.Writer, name string) {
	fmt.Fprintf(w, "# HELP %s Request latency in seconds\n", name)
	fmt.Fprintf(w, "# TYPE %s histogram\n", name)

	var cumulative int64
	for i, b := range h.buckets {
		cumulative += h.counts[i].Load()
		fmt.Fprintf(w, "%s_bucket{le=\"%g\"} %d\n", name, b, cumulative)
	}
	cumulative += h.counts[len(h.buckets)].Load()
	fmt.Fprintf(w, "%s_bucket{le=\"+Inf\"} %d\n", name, cumulative)
	fmt.Fprintf(w, "%s_sum %f\n", name, float64(h.sum.Load())/1e6)
	fmt.Fprintf(w, "%s_count %d\n", name, h.count.Load())

	// Per-path histograms
	h.mu.RLock()
	defer h.mu.RUnlock()
	for path, ph := range h.byPath {
		var cum int64
		for i, b := range ph.buckets {
			cum += ph.counts[i].Load()
			fmt.Fprintf(w, "%s_bucket{le=\"%g\",path=\"%s\"} %d\n", name, b, path, cum)
		}
		cum += ph.counts[len(ph.buckets)].Load()
		fmt.Fprintf(w, "%s_bucket{le=\"+Inf\",path=\"%s\"} %d\n", name, path, cum)
	}
}
