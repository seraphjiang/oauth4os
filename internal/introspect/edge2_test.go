package introspect

import (
	"sync"
	"testing"
	"time"
)

type edgeCountingLookup struct {
	mu    sync.Mutex
	calls int
}

func (c *edgeCountingLookup) Introspect(token string) *Response {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return &Response{Active: true, ClientID: "app"}
}

func TestEdge_CachedLookupHit(t *testing.T) {
	inner := &edgeCountingLookup{}
	c := NewCachedLookup(inner, time.Minute)
	c.Introspect("tok-1")
	c.Introspect("tok-1")
	inner.mu.Lock()
	n := inner.calls
	inner.mu.Unlock()
	if n != 1 {
		t.Errorf("cached should call inner once, got %d", n)
	}
}

func TestEdge_CachedLookupDifferentTokens(t *testing.T) {
	inner := &edgeCountingLookup{}
	c := NewCachedLookup(inner, time.Minute)
	c.Introspect("tok-1")
	c.Introspect("tok-2")
	inner.mu.Lock()
	n := inner.calls
	inner.mu.Unlock()
	if n != 2 {
		t.Errorf("different tokens should call inner twice, got %d", n)
	}
}

func TestEdge_CachedLookupConcurrent(t *testing.T) {
	inner := &edgeCountingLookup{}
	c := NewCachedLookup(inner, time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Introspect("tok-1")
		}()
	}
	wg.Wait()
}
