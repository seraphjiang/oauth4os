package session

import "testing"

func TestEdge_CreateDuplicateID(t *testing.T) {
	m := New(nil)
	m.Create("s1", "app", "t1", "1.2.3.4")
	ok := m.Create("s1", "app", "t2", "5.6.7.8")
	// Duplicate ID may overwrite or reject — just verify no panic
	_ = ok
}

func TestEdge_RemoveNonexistent(t *testing.T) {
	m := New(nil)
	m.Remove("never-created") // must not panic
}

func TestEdge_TouchNonexistent(t *testing.T) {
	m := New(nil)
	m.Touch("never-created") // must not panic
}

func TestEdge_ForceLogoutEmpty(t *testing.T) {
	m := New(nil)
	n := m.ForceLogout("no-sessions")
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestEdge_CountEmpty(t *testing.T) {
	m := New(nil)
	if c := m.Count("any"); c != 0 {
		t.Errorf("expected 0, got %d", c)
	}
}
