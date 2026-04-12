package e2e

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestStress_1000Concurrent sends 1000 concurrent requests through the proxy
// and reports p50/p95/p99 latency + throughput.
func TestStress_1000Concurrent(t *testing.T) {
	proxy := proxyURL(t)
	runStress(t, proxy, 1000, 50)
}

// BenchmarkProxyThroughput measures sustained throughput.
func BenchmarkProxyThroughput(b *testing.B) {
	proxy := stressProxyURL(b)
	client := &http.Client{Timeout: 10 * time.Second}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(proxy + "/health")
			if err == nil {
				resp.Body.Close()
			}
		}
	})
}

func runStress(t *testing.T, proxy string, total, concurrency int) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}

	latencies := make([]time.Duration, total)
	var errors atomic.Int64
	var mu sync.Mutex
	idx := 0

	// Endpoints to hit (mix of fast + auth paths)
	endpoints := []string{
		"/health",
		"/.well-known/openid-configuration",
		"/.well-known/jwks.json",
		"/oauth/register",
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < total; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(n int) {
			defer wg.Done()
			defer func() { <-sem }()

			ep := endpoints[n%len(endpoints)]
			method := "GET"
			if ep == "/oauth/register" {
				method = "GET" // list, not create
			}

			reqStart := time.Now()
			req, _ := http.NewRequest(method, proxy+ep, nil)
			resp, err := client.Do(req)
			elapsed := time.Since(reqStart)

			if err != nil {
				errors.Add(1)
				return
			}
			resp.Body.Close()

			if resp.StatusCode >= 500 {
				errors.Add(1)
			}

			mu.Lock()
			if idx < total {
				latencies[idx] = elapsed
				idx++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	wallTime := time.Since(start)

	// Trim to actual count
	mu.Lock()
	latencies = latencies[:idx]
	mu.Unlock()

	if len(latencies) == 0 {
		t.Fatal("no successful requests")
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	p50 := percentile(latencies, 50)
	p95 := percentile(latencies, 95)
	p99 := percentile(latencies, 99)
	errCount := errors.Load()
	throughput := float64(total) / wallTime.Seconds()

	t.Logf("")
	t.Logf("═══════════════════════════════════════")
	t.Logf("  Stress Test Results")
	t.Logf("═══════════════════════════════════════")
	t.Logf("  Requests:    %d total, %d concurrent", total, concurrency)
	t.Logf("  Successful:  %d", len(latencies))
	t.Logf("  Errors:      %d (%.1f%%)", errCount, float64(errCount)/float64(total)*100)
	t.Logf("  Wall time:   %s", wallTime.Round(time.Millisecond))
	t.Logf("  Throughput:  %.0f req/s", throughput)
	t.Logf("  Latency p50: %s", p50.Round(time.Microsecond))
	t.Logf("  Latency p95: %s", p95.Round(time.Microsecond))
	t.Logf("  Latency p99: %s", p99.Round(time.Microsecond))
	t.Logf("═══════════════════════════════════════")

	// Fail if error rate > 5%
	if float64(errCount)/float64(total) > 0.05 {
		t.Errorf("error rate %.1f%% exceeds 5%% threshold", float64(errCount)/float64(total)*100)
	}
	// Fail if p99 > 5s
	if p99 > 5*time.Second {
		t.Errorf("p99 latency %s exceeds 5s threshold", p99)
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func stressProxyURL(b *testing.B) string {
	b.Helper()
	for _, u := range []string{
		"https://f5cmk2hxwx.us-west-2.awsapprunner.com",
		"http://localhost:8443",
	} {
		resp, err := http.Get(u + "/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return u
		}
	}
	b.Skip("no proxy available")
	return ""
}

// proxyURL is defined in webapp_flow_test.go — this is a compile guard
var _ = fmt.Sprintf
