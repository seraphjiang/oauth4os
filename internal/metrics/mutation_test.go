package metrics

import "testing"

// Mutation: remove Inc → counter must increment
func TestMutation_CounterInc(t *testing.T) {
	c := NewCounter()
	l := Labels{Method: "GET", Path: "/health", Status: 200}
	c.Inc(l)
	c.Inc(l)
	if c.Get(l) != 2 {
		t.Errorf("expected 2, got %d", c.Get(l))
	}
}

// Mutation: remove label isolation → different labels must track independently
func TestMutation_LabelIsolation(t *testing.T) {
	c := NewCounter()
	c.Inc(Labels{Method: "GET", Path: "/a", Status: 200})
	c.Inc(Labels{Method: "POST", Path: "/b", Status: 201})
	if c.Get(Labels{Method: "GET", Path: "/a", Status: 200}) != 1 {
		t.Error("labels must be isolated")
	}
}
