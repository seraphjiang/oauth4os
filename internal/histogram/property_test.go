package histogram

import (
	"sync"
	"testing"
	"time"
)

// Property: concurrent Observe must produce accurate total count and sum
func TestProperty_ConcurrentAccuracy(t *testing.T) {
	h := New()
	n := 100
	dur := 10 * time.Millisecond
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.Observe(dur, "/test")
		}()
	}
	wg.Wait()

	if got := h.count.Load(); got != int64(n) {
		t.Errorf("count: expected %d, got %d", n, got)
	}
	expectedMicros := int64(n) * dur.Microseconds()
	gotMicros := h.sum.Load()
	// Allow 10% tolerance for timing
	if gotMicros < expectedMicros*9/10 || gotMicros > expectedMicros*11/10 {
		t.Errorf("sum: expected ~%dμs, got %dμs", expectedMicros, gotMicros)
	}
}
