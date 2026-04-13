package loadtest

import "testing"

func TestEdge_NewHarness(t *testing.T) {
	h := New("http://localhost:1", 1, 1)
	if h.BaseURL != "http://localhost:1" {
		t.Error("BaseURL should match")
	}
	if h.Clients != 1 || h.Iterations != 1 {
		t.Error("Clients and Iterations should match")
	}
}
