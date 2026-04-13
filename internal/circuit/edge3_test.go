package circuit

import (
	"testing"
	"time"
)

func TestEdge_SuccessResetsFailures(t *testing.T) {
	b := New(3, time.Minute)
	b.Record(500)
	b.Record(500)
	b.Record(200) // should reset
	b.Record(500)
	b.Record(500)
	// Should still be closed — success reset the counter
	if b.State() != Closed {
		t.Error("success should reset failure counter")
	}
}

func TestEdge_4xxNotCounted(t *testing.T) {
	b := New(2, time.Minute)
	b.Record(400)
	b.Record(404)
	b.Record(429)
	// 4xx should not trip the breaker
	if b.State() != Closed {
		t.Error("4xx should not count as failures")
	}
}
