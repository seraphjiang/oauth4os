package idempotency

import (
	"testing"
	"time"
)

func TestEdge_StopIdempotent(t *testing.T) {
	s := New(time.Minute)
	s.Stop()
	s.Stop() // must not panic
}

func TestEdge_ZeroTTLStopClean(t *testing.T) {
	s := New(0)
	time.Sleep(10 * time.Millisecond)
	s.Stop()
}

func TestEdge_LargeTTL(t *testing.T) {
	s := New(24 * time.Hour)
	defer s.Stop()
	// Should not block or consume resources
}

func TestEdge_NewReturnsNonNil(t *testing.T) {
	s := New(time.Second)
	defer s.Stop()
	if s == nil {
		t.Error("New should return non-nil")
	}
}
