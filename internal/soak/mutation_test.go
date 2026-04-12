package soak

import "testing"

// Mutation: remove String → Result must format readable output
func TestMutation_ResultString(t *testing.T) {
	r := Result{Requests: 100, StartHeap: 1.0, EndHeap: 2.0, HeapGrowth: 1.0, StartGR: 5, EndGR: 5}
	s := r.String()
	if s == "" {
		t.Error("String must produce output")
	}
	if len(s) < 10 {
		t.Errorf("String too short: %q", s)
	}
}
