package circuit

import (
	"sync"
	"testing"
	"time"
)

// Property: circuit breaker state machine transitions are consistent
// closed → open (after threshold failures) → half-open (after timeout) → closed (on success)
func TestProperty_StateMachine(t *testing.T) {
	for trial := 0; trial < 20; trial++ {
		b := New(3, 50*time.Millisecond)

		// Closed: should allow
		if !b.Allow() {
			t.Fatal("new breaker should be closed (allow)")
		}

		// Trip it: 3 failures
		for i := 0; i < 3; i++ {
			b.Record(500)
		}

		// Open: should reject
		if b.Allow() {
			t.Fatal("breaker should be open after 3 failures")
		}

		// Wait for half-open
		time.Sleep(60 * time.Millisecond)

		// Half-open: should allow one probe
		if !b.Allow() {
			t.Fatal("breaker should be half-open after timeout")
		}

		// Success closes it
		b.Record(200)

		// Closed again
		if !b.Allow() {
			t.Fatal("breaker should be closed after successful probe")
		}
	}
}

// Property: concurrent Record calls must not corrupt state
func TestProperty_ConcurrentRecord(t *testing.T) {
	b := New(1000000, time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if n%2 == 0 {
					b.Record(200)
				} else {
					b.Record(500)
				}
				b.Allow()
			}
		}(i)
	}
	wg.Wait()
}
