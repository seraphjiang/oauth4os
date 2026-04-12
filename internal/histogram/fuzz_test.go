package histogram

import (
	"bytes"
	"testing"
	"time"
)

func FuzzObserve(f *testing.F) {
	f.Add(int64(0), "/health")
	f.Add(int64(1000), "/search")
	f.Add(int64(999999999), "/_cat/indices")
	f.Add(int64(-1), "")
	f.Add(int64(5000000), "/very/deep/path/here")

	f.Fuzz(func(t *testing.T, micros int64, path string) {
		h := New()
		d := time.Duration(micros) * time.Microsecond
		h.Observe(d, path) // must not panic
		h.Observe(d, path)
		var buf bytes.Buffer
		h.WritePrometheus(&buf, "fuzz") // must not panic
		if h.count.Load() != 2 {
			t.Fatalf("expected count 2, got %d", h.count.Load())
		}
	})
}
