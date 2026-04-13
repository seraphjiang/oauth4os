package session

import "testing"

func TestEdge_CreateReturnsTrueUnderLimit(t *testing.T) {
	m := New(map[string]int{"*": 10})
	if !m.Create("s1", "app", "t1", "1.2.3.4") {
		t.Error("should succeed under limit")
	}
}

func TestEdge_CountAfterRemove(t *testing.T) {
	m := New(nil)
	m.Create("s1", "app", "t1", "1.2.3.4")
	m.Create("s2", "app", "t2", "1.2.3.4")
	m.Remove("s1")
	if c := m.Count("app"); c != 1 {
		t.Errorf("expected 1 after remove, got %d", c)
	}
}

func TestEdge_ListFiltersByClient(t *testing.T) {
	m := New(nil)
	m.Create("s1", "a", "t1", "1.2.3.4")
	m.Create("s2", "b", "t2", "1.2.3.4")
	list := m.List("a")
	if len(list) != 1 {
		t.Errorf("expected 1 for client a, got %d", len(list))
	}
}
