package soak

import (
	"testing"
	"time"
)

func TestEdge_ResultStringNotEmpty(t *testing.T) {
	r := Result{Requests: 50, StartHeap: 1.0, EndHeap: 2.0, HeapGrowth: 1.0, StartGR: 5, EndGR: 5}
	s := r.String()
	if len(s) < 10 {
		t.Errorf("String too short: %q", s)
	}
}

func TestEdge_ResultZeroRequests(t *testing.T) {
	r := Result{Duration: time.Second}
	s := r.String()
	if s == "" {
		t.Error("zero requests should still produce output")
	}
}
