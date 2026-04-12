package cache

import (
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	c := New(1*time.Second, 100)
	c.Set("/test", 200, map[string]string{"X-Cached": "true"}, []byte("hello"))
	e := c.Get("/test")
	if e == nil {
		t.Fatal("expected cached entry")
	}
	if string(e.Body) != "hello" || e.StatusCode != 200 {
		t.Errorf("unexpected entry: %+v", e)
	}
}

func TestMiss(t *testing.T) {
	c := New(1*time.Second, 100)
	if c.Get("/missing") != nil {
		t.Error("expected nil for missing key")
	}
}

func TestExpiry(t *testing.T) {
	c := New(50*time.Millisecond, 100)
	c.Set("/exp", 200, nil, []byte("data"))
	time.Sleep(100 * time.Millisecond)
	if c.Get("/exp") != nil {
		t.Error("expected expired entry to return nil")
	}
}

func TestEviction(t *testing.T) {
	c := New(1*time.Second, 2)
	c.Set("/a", 200, nil, []byte("a"))
	c.Set("/b", 200, nil, []byte("b"))
	c.Set("/c", 200, nil, []byte("c")) // should evict /a
	if c.Get("/c") == nil {
		t.Error("/c should exist")
	}
	// At least one of a/b should be evicted
	count := 0
	if c.Get("/a") != nil {
		count++
	}
	if c.Get("/b") != nil {
		count++
	}
	if count > 1 {
		t.Error("expected at least one eviction")
	}
}
