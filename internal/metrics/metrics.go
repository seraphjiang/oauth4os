// Package metrics provides Prometheus-compatible labeled metrics with cardinality guard.
package metrics

import (
	"fmt"
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// MaxCardinality caps unique label combinations per metric.
const MaxCardinality = 1000

// Labels is a set of key-value pairs identifying a time series.
type Labels struct {
	Method string
	Path   string
	Status int
}

func (l Labels) String() string {
	return fmt.Sprintf("method=%q,path=%q,status_code=%q", l.Method, l.Path, statusStr(l.Status))
}

func statusStr(code int) string {
	if code == 0 {
		return ""
	}
	return fmt.Sprintf("%d", code)
}

// Counter is a labeled monotonic counter.
type Counter struct {
	mu     sync.RWMutex
	series map[Labels]*atomic.Int64
}

func NewCounter() *Counter {
	return &Counter{series: make(map[Labels]*atomic.Int64)}
}

func (c *Counter) Inc(l Labels) {
	c.Add(l, 1)
}

func (c *Counter) Add(l Labels, n int64) {
	c.mu.RLock()
	v, ok := c.series[l]
	c.mu.RUnlock()
	if ok {
		v.Add(n)
		return
	}
	c.mu.Lock()
	if v, ok = c.series[l]; ok {
		c.mu.Unlock()
		v.Add(n)
		return
	}
	if len(c.series) >= MaxCardinality {
		c.mu.Unlock()
		return // drop — cardinality exceeded
	}
	v = &atomic.Int64{}
	v.Add(n)
	c.series[l] = v
	c.mu.Unlock()
}

func (c *Counter) WritePrometheus(w io.Writer, name, help string) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n", name, help, name)
	c.mu.RLock()
	defer c.mu.RUnlock()
	for l, v := range c.series {
		fmt.Fprintf(w, "%s{%s} %d\n", name, l, v.Load())
	}
}

// Summary tracks count, sum, min, max for a labeled metric (no quantiles — use histogram for that).
type Summary struct {
	mu     sync.RWMutex
	series map[Labels]*summaryData
}

type summaryData struct {
	count atomic.Int64
	sum   atomic.Int64 // microseconds
	min   atomic.Int64
	max   atomic.Int64
}

func NewSummary() *Summary {
	return &Summary{series: make(map[Labels]*summaryData)}
}

func (s *Summary) Observe(l Labels, d time.Duration) {
	us := d.Microseconds()
	s.mu.RLock()
	sd, ok := s.series[l]
	s.mu.RUnlock()
	if !ok {
		s.mu.Lock()
		sd, ok = s.series[l]
		if !ok {
			if len(s.series) >= MaxCardinality {
				s.mu.Unlock()
				return
			}
			sd = &summaryData{}
			sd.min.Store(math.MaxInt64)
			s.series[l] = sd
		}
		s.mu.Unlock()
	}
	sd.count.Add(1)
	sd.sum.Add(us)
	for {
		old := sd.min.Load()
		if us >= old || sd.min.CompareAndSwap(old, us) {
			break
		}
	}
	for {
		old := sd.max.Load()
		if us <= old || sd.max.CompareAndSwap(old, us) {
			break
		}
	}
}

func (s *Summary) WritePrometheus(w io.Writer, name, help string) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s summary\n", name, help, name)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for l, sd := range s.series {
		fmt.Fprintf(w, "%s_count{%s} %d\n", name, l, sd.count.Load())
		fmt.Fprintf(w, "%s_sum{%s} %f\n", name, l, float64(sd.sum.Load())/1e6)
	}
}

// Cardinality returns the number of unique label combinations.
func (c *Counter) Cardinality() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.series)
}

func (s *Summary) Cardinality() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.series)
}
