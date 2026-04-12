package histogram

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestObserveAndWrite(t *testing.T) {
	h := New()
	h.Observe(5*time.Millisecond, "/search")
	h.Observe(50*time.Millisecond, "/search")
	h.Observe(500*time.Millisecond, "/health")

	var buf bytes.Buffer
	h.WritePrometheus(&buf, "test_latency")
	out := buf.String()

	if !strings.Contains(out, "test_latency_bucket") {
		t.Fatal("expected bucket lines")
	}
	if !strings.Contains(out, "test_latency_count 3") {
		t.Fatalf("expected count 3, got:\n%s", out)
	}
	if !strings.Contains(out, `path="/search"`) {
		t.Fatal("expected per-path breakdown")
	}
	if !strings.Contains(out, `path="/health"`) {
		t.Fatal("expected /health path")
	}
}

func TestEmptyHistogram(t *testing.T) {
	h := New()
	var buf bytes.Buffer
	h.WritePrometheus(&buf, "empty")
	if !strings.Contains(buf.String(), "empty_count 0") {
		t.Fatal("empty histogram should have count 0")
	}
}

func BenchmarkObserve(b *testing.B) {
	h := New()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			h.Observe(10*time.Millisecond, "/test")
		}
	})
}
