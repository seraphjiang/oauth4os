// Package healthcheck provides background upstream health monitoring.
package healthcheck

import (
	"net/http"
	"sync"
	"time"
)

// Status holds the latest health check result.
type Status struct {
	Healthy  bool          `json:"healthy"`
	Latency  time.Duration `json:"latency_ms"`
	LastCheck time.Time    `json:"last_check"`
	Error    string        `json:"error,omitempty"`
}

// Checker pings the upstream periodically and exposes the latest status.
type Checker struct {
	mu       sync.RWMutex
	status   Status
	client   *http.Client
	url      string
	interval time.Duration
	stop     chan struct{}
	stopOnce sync.Once
}

// New starts a background health checker that pings url every interval.
func New(url string, interval time.Duration, transport http.RoundTripper) *Checker {
	c := &Checker{
		url:      url,
		interval: interval,
		client:   &http.Client{Timeout: 5 * time.Second, Transport: transport},
		stop:     make(chan struct{}),
	}
	go c.run()
	return c
}

// Status returns the latest health check result.
func (c *Checker) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

// Stop halts the background checker.
func (c *Checker) Stop() {
	c.stopOnce.Do(func() { close(c.stop) })
}

func (c *Checker) run() {
	c.check() // immediate first check
	if c.interval <= 0 {
		<-c.stop
		return
	}
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.check()
		case <-c.stop:
			return
		}
	}
}

func (c *Checker) check() {
	start := time.Now()
	resp, err := c.client.Get(c.url)
	latency := time.Since(start)

	s := Status{LastCheck: time.Now(), Latency: latency}
	if err != nil {
		s.Error = err.Error()
	} else {
		resp.Body.Close()
		s.Healthy = resp.StatusCode < 500
		if !s.Healthy {
			s.Error = http.StatusText(resp.StatusCode)
		}
	}

	c.mu.Lock()
	c.status = s
	c.mu.Unlock()
}
