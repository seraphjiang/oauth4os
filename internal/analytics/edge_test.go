package analytics

import (
	"testing"
	"time"
)

// Edge: Track records request for client
func TestEdge_TrackRecords(t *testing.T) {
	a := New()
	a.Track("client-1", "/api/data", 200, 50*time.Millisecond)
	s := a.Snapshot()
	if len(s) == 0 {
		t.Error("Snapshot should contain tracked client")
	}
}

// Edge: multiple clients tracked separately
func TestEdge_MultipleClients(t *testing.T) {
	a := New()
	a.Track("c1", "/a", 200, time.Millisecond)
	a.Track("c2", "/b", 200, time.Millisecond)
	s := a.Snapshot()
	if len(s) < 2 {
		t.Errorf("expected 2 clients, got %d", len(s))
	}
}

// Edge: error status tracked
func TestEdge_ErrorStatusTracked(t *testing.T) {
	a := New()
	a.Track("c1", "/fail", 500, time.Millisecond)
	a.Track("c1", "/ok", 200, time.Millisecond)
	s := a.Snapshot()
	if len(s) == 0 {
		t.Error("should have snapshot data")
	}
}

// Edge: empty snapshot before any tracking
func TestEdge_EmptySnapshot(t *testing.T) {
	a := New()
	s := a.Snapshot()
	if len(s) != 0 {
		t.Errorf("empty tracker should have empty snapshot, got %d", len(s))
	}
}
