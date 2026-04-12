// Package soak provides a continuous operation test to detect memory leaks and resource exhaustion.
package soak

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// Snapshot captures memory stats at a point in time.
type Snapshot struct {
	Time     time.Time
	HeapMB   float64
	Goroutines int
	Requests int64
}

// Result summarizes a soak test run.
type Result struct {
	Duration   time.Duration
	Requests   int64
	StartHeap  float64
	EndHeap    float64
	HeapGrowth float64 // MB
	StartGR    int
	EndGR      int
	GRGrowth   int
	Leaked     bool // true if heap grew >50% or goroutines grew >10
}

// Run executes a soak test: continuously hits the proxy for the given duration.
func Run(baseURL string, duration time.Duration, concurrency int) Result {
	startSnap := snapshot(0)
	var total atomic.Int64
	done := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		go func() {
			client := &http.Client{Timeout: 5 * time.Second}
			for {
				select {
				case <-done:
					return
				default:
					resp, err := client.Get(baseURL + "/health")
					if err == nil {
						resp.Body.Close()
					}
					total.Add(1)
				}
			}
		}()
	}

	time.Sleep(duration)
	close(done)
	time.Sleep(100 * time.Millisecond) // let goroutines drain

	// Force GC to get accurate heap
	runtime.GC()
	runtime.GC()

	endSnap := snapshot(total.Load())
	growth := endSnap.HeapMB - startSnap.HeapMB
	grGrowth := endSnap.Goroutines - startSnap.Goroutines

	return Result{
		Duration:   duration,
		Requests:   total.Load(),
		StartHeap:  startSnap.HeapMB,
		EndHeap:    endSnap.HeapMB,
		HeapGrowth: growth,
		StartGR:    startSnap.Goroutines,
		EndGR:      endSnap.Goroutines,
		GRGrowth:   grGrowth,
		Leaked:     growth > 50 || grGrowth > 10, // >50MB heap growth or >10 goroutine leak
	}
}

func snapshot(reqs int64) Snapshot {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return Snapshot{
		Time:       time.Now(),
		HeapMB:     float64(m.HeapAlloc) / 1024 / 1024,
		Goroutines: runtime.NumGoroutine(),
		Requests:   reqs,
	}
}

// String formats the result for display.
func (r Result) String() string {
	verdict := "PASS"
	if r.Leaked {
		verdict = "LEAK DETECTED"
	}
	return fmt.Sprintf("[%s] %s reqs=%d heap=%.1f→%.1fMB(+%.1f) goroutines=%d→%d(+%d)",
		verdict, r.Duration, r.Requests, r.StartHeap, r.EndHeap, r.HeapGrowth, r.StartGR, r.EndGR, r.GRGrowth)
}
