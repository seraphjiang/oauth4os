package circuit

import (
	"testing"
	"time"
)

func TestEdge_StateTransitions(t *testing.T) {
	b := New(2, 100*time.Millisecond)
	if b.State() != Closed {
		t.Error("initial state should be Closed")
	}
	// Trip the breaker
	b.Record(500)
	b.Record(500)
	if b.State() != Open {
		t.Error("after threshold failures, state should be Open")
	}
	// Wait for cooldown
	time.Sleep(150 * time.Millisecond)
	b.Allow() // triggers half-open
	if b.State() != HalfOpen {
		t.Error("after cooldown, Allow should move to HalfOpen")
	}
	// Success closes it
	b.Record(200)
	if b.State() != Closed {
		t.Error("success in HalfOpen should close breaker")
	}
}

func TestEdge_RetryAfterPositive(t *testing.T) {
	b := New(1, time.Minute)
	b.Record(500)
	ra := b.RetryAfter()
	if ra <= 0 {
		t.Errorf("RetryAfter should be positive when open, got %d", ra)
	}
}

func TestEdge_AllowWhenClosed(t *testing.T) {
	b := New(100, time.Minute)
	for i := 0; i < 50; i++ {
		if !b.Allow() {
			t.Error("closed breaker should always allow")
		}
	}
}
