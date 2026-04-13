package cache

import (
	"testing"
	"time"
)

func TestEdge_GetAfterStop(t *testing.T) {
	c := New(time.Minute, 100)
	c.Set("k", 200, nil, []byte("v"))
	c.Stop()
	// Get after stop should still work (data in memory)
	e := c.Get("k")
	if e == nil {
		t.Error("Get after Stop should still return cached data")
	}
}

func TestEdge_SetAfterStop(t *testing.T) {
	c := New(time.Minute, 100)
	c.Stop()
	c.Set("k", 200, nil, []byte("v"))
	// Should not panic
}

func TestEdge_HeadersPreserved(t *testing.T) {
	c := New(time.Minute, 100)
	defer c.Stop()
	hdrs := map[string]string{"Content-Type": "application/json", "X-Custom": "val"}
	c.Set("k", 200, hdrs, []byte("body"))
	e := c.Get("k")
	if e == nil {
		t.Fatal("should find entry")
	}
	if e.Header["Content-Type"] != "application/json" {
		t.Error("headers should be preserved")
	}
}
