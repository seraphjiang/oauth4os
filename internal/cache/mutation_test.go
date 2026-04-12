package cache

import (
	"testing"
	"time"
)

// Mutation: Get returns expired entries
func TestMutation_ExpiredReturned(t *testing.T) {
	c := New(10*time.Millisecond, 100)
	c.Set("/test", 200, nil, []byte("data"))
	time.Sleep(50 * time.Millisecond)
	if c.Get("/test") != nil {
		t.Error("MUTATION SURVIVED: expired entry should return nil")
	}
}

// Mutation: Set doesn't store anything
func TestMutation_SetNoOp(t *testing.T) {
	c := New(1*time.Second, 100)
	c.Set("/a", 200, nil, []byte("hello"))
	e := c.Get("/a")
	if e == nil {
		t.Error("MUTATION SURVIVED: Set should store the entry")
	}
	if string(e.Body) != "hello" {
		t.Error("MUTATION SURVIVED: stored body doesn't match")
	}
}

// Mutation: eviction disabled (maxSize ignored)
func TestMutation_EvictionDisabled(t *testing.T) {
	c := New(1*time.Second, 1)
	c.Set("/a", 200, nil, []byte("a"))
	c.Set("/b", 200, nil, []byte("b"))
	// With maxSize=1, only one should survive
	count := 0
	if c.Get("/a") != nil {
		count++
	}
	if c.Get("/b") != nil {
		count++
	}
	if count > 1 {
		t.Error("MUTATION SURVIVED: maxSize=1 but both entries exist")
	}
	if count == 0 {
		t.Error("MUTATION SURVIVED: at least one entry should exist")
	}
}

// Mutation: StatusCode not stored
func TestMutation_StatusCodeLost(t *testing.T) {
	c := New(1*time.Second, 100)
	c.Set("/test", 404, nil, []byte("not found"))
	e := c.Get("/test")
	if e == nil {
		t.Fatal("entry should exist")
	}
	if e.StatusCode != 404 {
		t.Errorf("MUTATION SURVIVED: expected 404, got %d", e.StatusCode)
	}
}
