package loadtest

import (
	"testing"
	"time"
)

// M1: New sets fields correctly.
func TestMutation_NewHarness(t *testing.T) {
	h := New("http://localhost:8443", 10, 100)
	if h.BaseURL != "http://localhost:8443" {
		t.Fatalf("BaseURL: got %s", h.BaseURL)
	}
	if h.Clients != 10 {
		t.Fatalf("Clients: got %d", h.Clients)
	}
	if h.Iterations != 100 {
		t.Fatalf("Iterations: got %d", h.Iterations)
	}
}

// M2: sortDurations sorts ascending.
func TestMutation_SortDurations(t *testing.T) {
	d := []time.Duration{300, 100, 200}
	sortDurations(d)
	if d[0] != 100 || d[1] != 200 || d[2] != 300 {
		t.Fatalf("expected sorted, got %v", d)
	}
}

// M3: sortDurations with single element.
func TestMutation_SortDurationsSingle(t *testing.T) {
	d := []time.Duration{42}
	sortDurations(d)
	if d[0] != 42 {
		t.Fatal("single element should be unchanged")
	}
}

// M4: percentile on known data.
func TestMutation_Percentile(t *testing.T) {
	h := New("http://x", 1, 1)
	h.results = make([]Result, 100)
	for i := 0; i < 100; i++ {
		h.results[i] = Result{Duration: time.Duration(i+1) * time.Millisecond}
	}
	p50 := h.percentile(50)
	if p50 < 49*time.Millisecond || p50 > 51*time.Millisecond {
		t.Fatalf("p50 expected ~50ms, got %v", p50)
	}
	p99 := h.percentile(99)
	if p99 < 98*time.Millisecond || p99 > 100*time.Millisecond {
		t.Fatalf("p99 expected ~99ms, got %v", p99)
	}
}
