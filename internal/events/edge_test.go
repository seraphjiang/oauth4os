package events

import "testing"

func TestEdge_NewNotifier(t *testing.T) {
	n := New([]string{"http://localhost:1/webhook"})
	if n == nil {
		t.Error("New should return non-nil")
	}
	n.Stop()
}

func TestEdge_EmitAfterStop(t *testing.T) {
	n := New([]string{"http://localhost:1/webhook"})
	n.Stop()
	// Emit after stop should not panic
	n.Emit(Event{Type: TokenIssued, ClientID: "app"})
}
