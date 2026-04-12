package analytics

import "testing"

// Mutation: remove Record → Snapshot must reflect recorded data
func TestMutation_RecordAndSnapshot(t *testing.T) {
	tr := New()
	tr.Record("app", []string{"read"}, "logs-2024")
	tr.Record("app", []string{"read"}, "logs-2024")
	tr.Record("other", nil, "metrics")
	snap := tr.Snapshot()
	if snap.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", snap.TotalRequests)
	}
}

// Mutation: remove per-client tracking → must track by client
func TestMutation_PerClient(t *testing.T) {
	tr := New()
	tr.Record("app-a", nil, "logs")
	tr.Record("app-b", nil, "logs")
	snap := tr.Snapshot()
	if len(snap.ByClient) < 2 {
		t.Errorf("expected 2 clients, got %d", len(snap.ByClient))
	}
}

// Mutation: remove per-index tracking → must track by index
func TestMutation_PerIndex(t *testing.T) {
	tr := New()
	tr.Record("app", nil, "logs-a")
	tr.Record("app", nil, "logs-b")
	snap := tr.Snapshot()
	if len(snap.ByIndex) < 2 {
		t.Errorf("expected 2 indices, got %d", len(snap.ByIndex))
	}
}
