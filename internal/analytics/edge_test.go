package analytics

import "testing"

// Edge: Record + Snapshot round-trip
func TestEdge_RecordSnapshot(t *testing.T) {
	a := New()
	a.Record("c1", []string{"read"}, "logs")
	s := a.Snapshot()
	if len(s.Clients) == 0 {
		t.Error("Snapshot should contain tracked client")
	}
}

// Edge: multiple clients tracked
func TestEdge_MultipleClients(t *testing.T) {
	a := New()
	a.Record("c1", []string{"read"}, "logs")
	a.Record("c2", []string{"write"}, "metrics")
	s := a.Snapshot()
	if len(s.Clients) < 2 {
		t.Errorf("expected 2 clients, got %d", len(s.Clients))
	}
}

// Edge: empty snapshot
func TestEdge_EmptySnapshot(t *testing.T) {
	a := New()
	s := a.Snapshot()
	if len(s.Clients) != 0 {
		t.Errorf("empty tracker should have 0 clients, got %d", len(s.Clients))
	}
}
