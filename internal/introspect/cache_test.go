package introspect

import (
	"sync/atomic"
	"testing"
	"time"
)

type countingLookup struct {
	calls atomic.Int64
	resp  *Response
}

func (c *countingLookup) Introspect(token string) *Response {
	c.calls.Add(1)
	return c.resp
}

func TestCacheHit(t *testing.T) {
	inner := &countingLookup{resp: &Response{Active: true, ClientID: "svc-1"}}
	cached := NewCachedLookup(inner, 1*time.Second)

	// First call — cache miss
	r1 := cached.Introspect("tok-1")
	if !r1.Active || inner.calls.Load() != 1 {
		t.Fatal("first call should hit inner")
	}

	// Second call — cache hit
	r2 := cached.Introspect("tok-1")
	if !r2.Active || inner.calls.Load() != 1 {
		t.Fatal("second call should be cached")
	}
}

func TestCacheExpiry(t *testing.T) {
	inner := &countingLookup{resp: &Response{Active: true}}
	cached := NewCachedLookup(inner, 50*time.Millisecond)

	cached.Introspect("tok-1")
	time.Sleep(60 * time.Millisecond)
	cached.Introspect("tok-1")

	if inner.calls.Load() != 2 {
		t.Fatalf("expected 2 inner calls after expiry, got %d", inner.calls.Load())
	}
}

func TestCacheDifferentTokens(t *testing.T) {
	inner := &countingLookup{resp: &Response{Active: true}}
	cached := NewCachedLookup(inner, 1*time.Second)

	cached.Introspect("tok-1")
	cached.Introspect("tok-2")

	if inner.calls.Load() != 2 {
		t.Fatalf("different tokens should each call inner, got %d", inner.calls.Load())
	}
}

func TestCacheInactiveToken(t *testing.T) {
	inner := &countingLookup{resp: &Response{Active: false}}
	cached := NewCachedLookup(inner, 1*time.Second)

	r := cached.Introspect("revoked-tok")
	if r.Active {
		t.Fatal("should cache inactive response too")
	}
	// Second call should be cached
	cached.Introspect("revoked-tok")
	if inner.calls.Load() != 1 {
		t.Fatal("inactive tokens should also be cached")
	}
}
