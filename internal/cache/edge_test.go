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
	c.Set("k1", 200, nil, []byte("hello"))
	e := c.Get("k1")
	if e == nil || e.StatusCode != 200 {
		t.Error("Set+Get should round-trip")
	}
}

// Edge: max size eviction
func TestEdge_MaxSizeEviction(t *testing.T) {
	c := New(time.Minute, 2)
	defer c.Stop()
	c.Set("k1", 200, nil, []byte("a"))
	c.Set("k2", 200, nil, []byte("b"))
	c.Set("k3", 200, nil, []byte("c"))
	if e := c.Get("k3"); e == nil {
		t.Error("newest entry should be retrievable")
	}
}
