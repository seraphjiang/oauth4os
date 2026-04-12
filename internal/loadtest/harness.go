// Package loadtest provides a harness for concurrent OAuth flow load testing.
package loadtest

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Result captures a single request outcome.
type Result struct {
	Status   int
	Duration time.Duration
	Err      error
}

// Report summarizes a load test run.
type Report struct {
	Total    int64
	Success  int64
	Errors   int64
	Duration time.Duration
	P50      time.Duration
	P95      time.Duration
	P99      time.Duration
	RPS      float64
}

// Harness runs concurrent OAuth flows against a proxy.
type Harness struct {
	BaseURL    string
	Clients    int
	Iterations int
	results    []Result
	mu         sync.Mutex
}

// New creates a load test harness.
func New(baseURL string, clients, iterations int) *Harness {
	return &Harness{BaseURL: baseURL, Clients: clients, Iterations: iterations}
}

// Run executes the load test: each client registers, gets a token, makes authenticated requests.
func (h *Harness) Run() Report {
	start := time.Now()
	var wg sync.WaitGroup
	var total, success, errors atomic.Int64

	for c := 0; c < h.Clients; c++ {
		wg.Add(1)
		go func(clientNum int) {
			defer wg.Done()
			client := &http.Client{Timeout: 10 * time.Second}
			clientID := fmt.Sprintf("load-client-%d", clientNum)

			for i := 0; i < h.Iterations; i++ {
				total.Add(1)
				r := h.doFlow(client, clientID)
				h.mu.Lock()
				h.results = append(h.results, r)
				h.mu.Unlock()
				if r.Err != nil || r.Status >= 400 {
					errors.Add(1)
				} else {
					success.Add(1)
				}
			}
		}(c)
	}
	wg.Wait()
	elapsed := time.Since(start)

	return Report{
		Total:    total.Load(),
		Success:  success.Load(),
		Errors:   errors.Load(),
		Duration: elapsed,
		P50:      h.percentile(50),
		P95:      h.percentile(95),
		P99:      h.percentile(99),
		RPS:      float64(total.Load()) / elapsed.Seconds(),
	}
}

func (h *Harness) doFlow(client *http.Client, clientID string) Result {
	start := time.Now()

	// Step 1: Get token via client_credentials
	body := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {"secret"},
		"scope":         {"read:logs-*"},
	}.Encode()

	resp, err := client.Post(h.BaseURL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(body))
	if err != nil {
		return Result{Err: err, Duration: time.Since(start)}
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	return Result{Status: resp.StatusCode, Duration: time.Since(start)}
}

func (h *Harness) percentile(p int) time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.results) == 0 {
		return 0
	}
	// Simple sorted percentile
	durations := make([]time.Duration, len(h.results))
	for i, r := range h.results {
		durations[i] = r.Duration
	}
	sortDurations(durations)
	idx := len(durations) * p / 100
	if idx >= len(durations) {
		idx = len(durations) - 1
	}
	return durations[idx]
}

func sortDurations(d []time.Duration) {
	for i := 1; i < len(d); i++ {
		for j := i; j > 0 && d[j] < d[j-1]; j-- {
			d[j], d[j-1] = d[j-1], d[j]
		}
	}
}
