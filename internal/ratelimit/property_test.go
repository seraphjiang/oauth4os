package ratelimit

import (
	"strconv"
	"sync"
	"testing"
)

// Property: concurrent Allow calls must not panic or corrupt state
func TestProperty_ConcurrentAllow(t *testing.T) {
	l := New(nil, 10000)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				l.Allow("client-"+strconv.Itoa(n%5), nil)
			}
		}(i)
	}
	wg.Wait()
}

// Property: different clients never interfere with each other
func TestProperty_ClientIsolation(t *testing.T) {
	l := New(map[string]int{"limited": 1}, 10000)
	// Exhaust "limited"
	for l.Allow("limited", nil) {
	}
	// "unlimited" must still work
	for i := 0; i < 100; i++ {
		if !l.Allow("unlimited", nil) {
			t.Fatalf("unlimited client blocked at request %d", i+1)
		}
	}
}
