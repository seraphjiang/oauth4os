package audit

import (
	"testing"
)

// Edge: Log records action
func TestEdge_LogRecords(t *testing.T) {
	a := New(1000)
	a.Log("admin", "create_client", "app-1", "created new client")
	entries := a.List(10)
	if len(entries) == 0 {
		t.Error("Log should record entry")
	}
}

// Edge: List respects limit
func TestEdge_ListLimit(t *testing.T) {
	a := New(1000)
	for i := 0; i < 20; i++ {
		a.Log("admin", "action", "target", "detail")
	}
	entries := a.List(5)
	if len(entries) > 5 {
		t.Errorf("List(5) should return at most 5, got %d", len(entries))
	}
}

// Edge: empty audit log returns empty list
func TestEdge_EmptyList(t *testing.T) {
	a := New(1000)
	entries := a.List(10)
	if len(entries) != 0 {
		t.Errorf("empty log should return empty list, got %d", len(entries))
	}
}

// Edge: max entries enforced
func TestEdge_MaxEntries(t *testing.T) {
	a := New(5)
	for i := 0; i < 20; i++ {
		a.Log("admin", "action", "target", "detail")
	}
	entries := a.List(100)
	if len(entries) > 5 {
		t.Errorf("max 5 entries, got %d", len(entries))
	}
}
