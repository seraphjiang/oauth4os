package histogram

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

// Mutation: remove Observe → count must increment
func TestMutation_ObserveIncrementsCount(t *testing.T) {
	h := New()
	h.Observe(50*time.Millisecond, "/search")
	h.Observe(100*time.Millisecond, "/search")
	if h.count.Load() != 2 {
		t.Errorf("expected count 2, got %d", h.count.Load())
	}
}

// Mutation: remove bucket assignment → must land in correct bucket
func TestMutation_BucketPlacement(t *testing.T) {
	h := New()
	h.Observe(3*time.Millisecond, "") // ≤ 0.005s bucket
	if h.counts[0].Load() != 1 {
		t.Error("3ms should land in first bucket (≤5ms)")
	}
}

// Mutation: remove per-path tracking → WritePrometheus must include path labels
func TestMutation_PerPathTracking(t *testing.T) {
	h := New()
	h.Observe(10*time.Millisecond, "/api/query")
	h.Observe(20*time.Millisecond, "/api/health")
	var buf bytes.Buffer
	h.WritePrometheus(&buf, "http_request_duration")
	out := buf.String()
	if !strings.Contains(out, `path="/api/query"`) {
		t.Error("must include per-path histogram for /api/query")
	}
	if !strings.Contains(out, `path="/api/health"`) {
		t.Error("must include per-path histogram for /api/health")
	}
}

// Mutation: remove sum tracking → sum must reflect total latency
func TestMutation_SumTracking(t *testing.T) {
	h := New()
	h.Observe(100*time.Millisecond, "")
	h.Observe(200*time.Millisecond, "")
	sumMicros := h.sum.Load()
	if sumMicros < 250000 { // at least 250ms total
		t.Errorf("sum should be ≥250ms, got %dμs", sumMicros)
	}
}

// Mutation: remove WritePrometheus → must output valid Prometheus format
func TestMutation_PrometheusFormat(t *testing.T) {
	h := New()
	h.Observe(50*time.Millisecond, "/test")
	var buf bytes.Buffer
	h.WritePrometheus(&buf, "latency")
	out := buf.String()
	if !strings.Contains(out, "# TYPE latency histogram") {
		t.Error("must include TYPE header")
	}
	if !strings.Contains(out, `latency_bucket{le="+Inf"}`) {
		t.Error("must include +Inf bucket")
	}
	if !strings.Contains(out, "latency_count 1") {
		t.Error("must include count")
	}
}

// Property: concurrent Observe must not panic or lose counts
func TestProperty_ConcurrentObserve(t *testing.T) {
	h := New()
	var wg sync.WaitGroup
	n := 100
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.Observe(time.Millisecond, "/concurrent")
		}()
	}
	wg.Wait()
	if h.count.Load() != int64(n) {
		t.Errorf("expected %d, got %d", n, h.count.Load())
	}
}
