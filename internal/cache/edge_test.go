package cache

import (
	"testing"
	"time"
)

// Edge: Get returns nil for missing key
func TestEdge_GetMissing(t *testing.T) {
	c := New(time.Minute, 100)
	defer c.Stop()
	if e := c.Get("nonexistent"); e != nil {
		t.Error("missing key should return nil")
	}
}

// Edge: Set+Get round-trip
func TestEdge_SetGetRoundTrip(t *testing.T) {
	c := New(time.Minute, 100)
	defer c.Stop()
	c.Set("k1", &Entry{Body: []byte("hello"), StatusCode: 200})
	e := c.Get("k1")
	if e == nil || e.StatusCode != 200 {
		t.Error("Set+Get should round-trip")
	}
}

// Edge: expired entry returns nil
func TestEdge_ExpiredReturnsNil(t *testing.T) {
	c := New(time.Millisecond, 100)
	defer c.Stop()
	c.Set("k1", &Entry{Body: []byte("hello"), StatusCode: 200, ExpiresAt: time.Now().Add(-time.Hour)})
	if e := c.Get("k1"); e != nil {
		t.Error("expired entry should return nil")
	}
}

// Edge: max size eviction
func TestEdge_MaxSizeEviction(t *testing.T) {
	c := New(time.Minute, 2)
	defer c.Stop()
	c.Set("k1", &Entry{Body: []byte("a"), StatusCode: 200})
	c.Set("k2", &Entry{Body: []byte("b"), StatusCode: 200})
	c.Set("k3", &Entry{Body: []byte("c"), StatusCode: 200})
	// At least k3 should be retrievable
	if e := c.Get("k3"); e == nil {
		t.Error("newest entry should be retrievable")
	}
}
