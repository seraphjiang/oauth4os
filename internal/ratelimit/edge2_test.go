package ratelimit

import (
	"sync"
	"testing"
)

// Edge: concurrent Allow on same client is safe
func TestEdge_ConcurrentSameClient(t *testing.T) {
	l := New(nil, 6000)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.Allow("client-1", []string{"read"})
		}()
	}
	wg.Wait()
}

// Edge: RetryAfter for unknown client returns small value
func TestEdge_RetryAfterUnknown(t *testing.T) {
	l := New(nil, 600)
	ra := l.RetryAfter("never-seen")
	if ra < 0 {
		t.Errorf("RetryAfter should be non-negative, got %d", ra)
	}
}
