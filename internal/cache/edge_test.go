package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestEviction_OldestRemoved(t *testing.T) {
	c := New(time.Minute, 3) // max 3 entries
	c.Set("a", 200, nil, []byte("1"))
	c.Set("b", 200, nil, []byte("2"))
	c.Set("c", 200, nil, []byte("3"))
	c.Set("d", 200, nil, []byte("4")) // should evict oldest

	// One of a/b/c should be evicted, d should exist
	if c.Get("d") == nil {
		t.Fatal("newest entry should exist")
	}
	count := 0
	for _, k := range []string{"a", "b", "c"} {
		if c.Get(k) != nil {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected 2 of 3 old entries to survive, got %d", count)
	}
}

func TestExpiry_StaleNotReturned(t *testing.T) {
	c := New(10*time.Millisecond, 100)
	c.Set("k", 200, nil, []byte("data"))
	if c.Get("k") == nil {
		t.Fatal("should exist immediately")
	}
	time.Sleep(15 * time.Millisecond)
	if c.Get("k") != nil {
		t.Fatal("should be expired")
	}
}

func TestConcurrentSetEviction(t *testing.T) {
	c := New(time.Minute, 10)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.Set(fmt.Sprintf("k%d", i), 200, nil, []byte("v"))
		}(i)
	}
	wg.Wait()
	// Should not panic or corrupt — just verify some entries exist
	found := 0
	for i := 0; i < 100; i++ {
		if c.Get(fmt.Sprintf("k%d", i)) != nil {
			found++
		}
	}
	if found == 0 {
		t.Fatal("expected some entries to survive concurrent writes")
	}
	if found > 10 {
		t.Fatalf("max 10 entries, but found %d", found)
	}
}

func TestHeadersPreserved(t *testing.T) {
	c := New(time.Minute, 10)
	c.Set("k", 200, map[string]string{"Content-Type": "application/json", "X-Custom": "val"}, []byte("{}"))
	e := c.Get("k")
	if e == nil {
		t.Fatal("expected hit")
	}
	if e.Header["Content-Type"] != "application/json" {
		t.Fatalf("expected Content-Type header, got %v", e.Header)
	}
	if e.Header["X-Custom"] != "val" {
		t.Fatalf("expected X-Custom header, got %v", e.Header)
	}
}
