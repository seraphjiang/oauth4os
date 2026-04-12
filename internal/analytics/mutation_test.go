package analytics

import "testing"

// Mutation: remove Record → Snapshot must reflect recorded data
func TestMutation_RecordAndSnapshot(t *testing.T) {
	tr := New()
	tr.Record("app", []string{"read"}, "logs-2024")
	tr.Record("app", []string{"read"}, "logs-2024")
	snap := tr.Snapshot()
	if len(snap.Clients) == 0 {
		t.Error("snapshot must include client data after Record")
	}
}

// Mutation: remove per-index tracking → must track indices
func TestMutation_IndexTracking(t *testing.T) {
	tr := New()
	tr.Record("app", nil, "logs-a")
	tr.Record("app", nil, "logs-b")
	snap := tr.Snapshot()
	if len(snap.Indices) < 2 {
		t.Errorf("expected 2 indices, got %d", len(snap.Indices))
	}
}

// Mutation: remove scope tracking → must track scopes
func TestMutation_ScopeTracking(t *testing.T) {
	tr := New()
	tr.Record("app", []string{"read", "write"}, "logs")
	snap := tr.Snapshot()
	if len(snap.Scopes) == 0 {
		t.Error("snapshot must include scope distribution")
	}
}
