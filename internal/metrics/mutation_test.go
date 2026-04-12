package metrics

import "testing"

// Mutation: remove Inc → counter must increment
func TestMutation_CounterInc(t *testing.T) {
	c := NewCounter()
	l := Labels{Method: "GET", Path: "/health", Status: 200}
	c.Inc(l)
	c.Inc(l)
	// Verify via snapshot
	snap := c.Snapshot()
	found := false
	for _, e := range snap {
		if e.Labels.Method == "GET" && e.Count == 2 {
			found = true
		}
	}
	if !found {
		t.Error("counter must track increments")
	}
}

// Mutation: remove label isolation → different labels must track independently
func TestMutation_LabelIsolation(t *testing.T) {
	c := NewCounter()
	c.Inc(Labels{Method: "GET", Path: "/a", Status: 200})
	c.Inc(Labels{Method: "POST", Path: "/b", Status: 201})
	snap := c.Snapshot()
	if len(snap) < 2 {
		t.Errorf("expected 2 label combos, got %d", len(snap))
	}
}
