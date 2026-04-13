package plugin

import "testing"

// Edge: empty registry authorize passes
func TestEdge_EmptyRegistryPasses(t *testing.T) {
	reg := NewRegistry()
	err := reg.Authorize(nil, nil)
	if err != nil {
		t.Errorf("empty registry should pass, got %v", err)
	}
}

// Edge: List on empty registry returns empty
func TestEdge_EmptyListEmpty(t *testing.T) {
	reg := NewRegistry()
	names := reg.List()
	if len(names) != 0 {
		t.Errorf("empty registry should list 0, got %d", len(names))
	}
}
