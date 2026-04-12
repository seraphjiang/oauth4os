package metrics

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCounterLabeled(t *testing.T) {
	c := NewCounter()
	c.Inc(Labels{Method: "GET", Path: "/search", Status: 200})
	c.Inc(Labels{Method: "GET", Path: "/search", Status: 200})
	c.Inc(Labels{Method: "POST", Path: "/token", Status: 401})

	var buf bytes.Buffer
	c.WritePrometheus(&buf, "req_total", "Total requests")
	out := buf.String()
	if !strings.Contains(out, `method="GET"`) {
		t.Fatalf("expected method label:\n%s", out)
	}
	if !strings.Contains(out, `status_code="401"`) {
		t.Fatalf("expected status label:\n%s", out)
	}
	if c.Cardinality() != 2 {
		t.Fatalf("expected 2 series, got %d", c.Cardinality())
	}
}

func TestSummaryObserve(t *testing.T) {
	s := NewSummary()
	l := Labels{Method: "GET", Path: "/health"}
	s.Observe(l, 10*time.Millisecond)
	s.Observe(l, 20*time.Millisecond)

	var buf bytes.Buffer
	s.WritePrometheus(&buf, "latency", "Request latency")
	out := buf.String()
	if !strings.Contains(out, "latency_count") {
		t.Fatalf("expected count:\n%s", out)
	}
	if !strings.Contains(out, "latency_sum") {
		t.Fatalf("expected sum:\n%s", out)
	}
}

func TestCardinalityGuard(t *testing.T) {
	c := NewCounter()
	for i := 0; i < MaxCardinality+100; i++ {
		c.Inc(Labels{Method: "GET", Path: statusStr(i)})
	}
	if c.Cardinality() != MaxCardinality {
		t.Fatalf("expected cardinality capped at %d, got %d", MaxCardinality, c.Cardinality())
	}
}

func TestSummaryCardinalityGuard(t *testing.T) {
	s := NewSummary()
	for i := 0; i < MaxCardinality+100; i++ {
		s.Observe(Labels{Path: statusStr(i)}, time.Millisecond)
	}
	if s.Cardinality() != MaxCardinality {
		t.Fatalf("expected cardinality capped at %d, got %d", MaxCardinality, s.Cardinality())
	}
}

func BenchmarkCounterInc(b *testing.B) {
	c := NewCounter()
	l := Labels{Method: "GET", Path: "/search", Status: 200}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc(l)
		}
	})
}

func BenchmarkSummaryObserve(b *testing.B) {
	s := NewSummary()
	l := Labels{Method: "GET", Path: "/search", Status: 200}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Observe(l, 10*time.Millisecond)
		}
	})
}

func TestConcurrentCounter(t *testing.T) {
	c := NewCounter()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Inc(Labels{Method: "GET", Status: n % 5})
			}
		}(i)
	}
	wg.Wait()
}
