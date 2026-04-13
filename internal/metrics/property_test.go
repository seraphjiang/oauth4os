package metrics

import (
	"bytes"
	"sync"
	"testing"
)

// Property: concurrent Inc must not lose counts
func TestProperty_ConcurrentInc(t *testing.T) {
	c := NewCounter()
	l := Labels{Method: "GET", Path: "/test", Status: 200}
	n := 100
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Inc(l)
		}()
	}
	wg.Wait()
	// Verify via WritePrometheus
	var buf bytes.Buffer
	c.WritePrometheus(&buf, "test", "test")
	if !bytes.Contains(buf.Bytes(), []byte("100")) {
		t.Errorf("expected 100 in output, got: %s", buf.String()[:min(len(buf.String()), 200)])
	}
}
