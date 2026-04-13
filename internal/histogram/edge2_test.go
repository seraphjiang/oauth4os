package histogram

import (
	"sync"
	"testing"
	"time"
)

func TestEdge_ConcurrentObserve(t *testing.T) {
	h := New()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(d int) {
			defer wg.Done()
			h.Observe(time.Duration(d)*time.Millisecond, "/api")
		}(i)
	}
	wg.Wait()
}

func TestEdge_LargeValue(t *testing.T) {
	h := New()
	h.Observe(10*time.Second, "/slow")
	// Should land in highest bucket — no panic
}

func TestEdge_NegativeDuration(t *testing.T) {
	h := New()
	h.Observe(-1*time.Millisecond, "/weird")
	// Should not panic
}
