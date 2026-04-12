package histogram

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestInfBucket(t *testing.T) {
	h := New()
	h.Observe(30*time.Second, "/slow") // way above all buckets
	var buf bytes.Buffer
	h.WritePrometheus(&buf, "test")
	out := buf.String()
	if !strings.Contains(out, `le="+Inf"} 1`) {
		t.Fatalf("expected +Inf bucket count 1, got:\n%s", out)
	}
}

func TestCumulativeBuckets(t *testing.T) {
	h := New()
	// 3 observations: 1ms (bucket 0.005), 50ms (bucket 0.05), 20s (+Inf)
	h.Observe(1*time.Millisecond, "")
	h.Observe(50*time.Millisecond, "")
	h.Observe(20*time.Second, "")

	var buf bytes.Buffer
	h.WritePrometheus(&buf, "lat")
	out := buf.String()

	// le=0.005 should have 1 (the 1ms observation)
	if !strings.Contains(out, `le="0.005"} 1`) {
		t.Fatalf("expected le=0.005 count 1:\n%s", out)
	}
	// le=+Inf should have 3 (cumulative)
	if !strings.Contains(out, `le="+Inf"} 3`) {
		t.Fatalf("expected +Inf count 3:\n%s", out)
	}
	// count should be 3
	if !strings.Contains(out, "lat_count 3") {
		t.Fatalf("expected count 3:\n%s", out)
	}
}

func TestSumAccuracy(t *testing.T) {
	h := New()
	h.Observe(100*time.Millisecond, "")
	h.Observe(200*time.Millisecond, "")
	var buf bytes.Buffer
	h.WritePrometheus(&buf, "s")
	out := buf.String()
	// Sum should be ~0.3 seconds
	if !strings.Contains(out, "s_sum 0.3") {
		t.Fatalf("expected sum ~0.3:\n%s", out)
	}
}
