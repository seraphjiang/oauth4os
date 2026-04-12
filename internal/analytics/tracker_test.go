package analytics

import "testing"

func TestRecordAndSnapshot(t *testing.T) {
	tr := New()
	tr.Record("agent-1", []string{"read:logs-*"}, "logs-2026")
	tr.Record("agent-1", []string{"read:logs-*"}, "logs-2026")
	tr.Record("agent-2", []string{"write:metrics-*"}, "metrics-cpu")

	r := tr.Snapshot()

	if len(r.Clients) != 2 {
		t.Fatalf("clients = %d, want 2", len(r.Clients))
	}
	// agent-1 should be first (2 requests)
	if r.Clients[0].ClientID != "agent-1" || r.Clients[0].Requests != 2 {
		t.Errorf("top client = %+v", r.Clients[0])
	}

	if len(r.Scopes) != 2 {
		t.Fatalf("scopes = %d, want 2", len(r.Scopes))
	}
	if r.Scopes[0].Name != "read:logs-*" || r.Scopes[0].Count != 2 {
		t.Errorf("top scope = %+v", r.Scopes[0])
	}

	if len(r.Indices) != 2 {
		t.Fatalf("indices = %d, want 2", len(r.Indices))
	}
	if r.Indices[0].Name != "logs-2026" || r.Indices[0].Count != 2 {
		t.Errorf("top index = %+v", r.Indices[0])
	}
}

func TestEmptySnapshot(t *testing.T) {
	tr := New()
	r := tr.Snapshot()
	if len(r.Clients) != 0 || len(r.Scopes) != 0 || len(r.Indices) != 0 {
		t.Error("empty tracker should return empty report")
	}
}
