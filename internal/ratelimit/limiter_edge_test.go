package ratelimit

import (
	"sync"
	"testing"
)

func TestConcurrentAllow(t *testing.T) {
	l := New(nil, 1000)
	var wg sync.WaitGroup
	allowed := make([]bool, 200)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			allowed[n] = l.Allow("client-1", nil)
		}(i)
	}
	wg.Wait()

	count := 0
	for _, a := range allowed {
		if a {
			count++
		}
	}
	if count == 0 {
		t.Fatal("at least some requests should be allowed")
	}
}

func TestPerClientIsolation(t *testing.T) {
	l := New(nil, 5) // 5 RPM

	// Exhaust client-1
	for i := 0; i < 10; i++ {
		l.Allow("client-1", nil)
	}

	// client-2 should still be allowed
	if !l.Allow("client-2", nil) {
		t.Fatal("client-2 should have its own bucket")
	}
}

func TestRetryAfterPositive(t *testing.T) {
	l := New(nil, 1) // 1 RPM
	l.Allow("client-1", nil)
	l.Allow("client-1", nil) // should be denied

	ra := l.RetryAfter("client-1")
	if ra <= 0 {
		t.Fatalf("RetryAfter should be positive when rate limited, got %d", ra)
	}
}

func TestScopeBasedLimits(t *testing.T) {
	l := New(map[string]int{"admin": 10, "read:logs-*": 100}, 50)

	// Admin scope should get 10 RPM (most restrictive)
	for i := 0; i < 15; i++ {
		l.Allow("admin-client", []string{"admin"})
	}
	if l.Allow("admin-client", []string{"admin"}) {
		t.Fatal("admin should be rate limited at 10 RPM")
	}
}
