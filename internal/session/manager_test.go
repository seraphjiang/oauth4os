package session

import (
	"testing"
	"time"
)

func TestCreateAndList(t *testing.T) {
	m := New(nil)
	m.Create("s1", "client-a", "tok1", "10.0.0.1")
	m.Create("s2", "client-a", "tok2", "10.0.0.2")
	m.Create("s3", "client-b", "tok3", "10.0.0.3")

	if got := len(m.List("")); got != 3 {
		t.Errorf("total = %d, want 3", got)
	}
	if got := len(m.List("client-a")); got != 2 {
		t.Errorf("client-a = %d, want 2", got)
	}
}

func TestSessionLimit(t *testing.T) {
	m := New(map[string]int{"limited": 2, "*": 100})

	if !m.Create("s1", "limited", "t1", "") {
		t.Error("first session should succeed")
	}
	if !m.Create("s2", "limited", "t2", "") {
		t.Error("second session should succeed")
	}
	if m.Create("s3", "limited", "t3", "") {
		t.Error("third session should be rejected (limit=2)")
	}
	// Other clients use global limit
	if !m.Create("s4", "other", "t4", "") {
		t.Error("other client should succeed")
	}
}

func TestForceLogout(t *testing.T) {
	m := New(nil)
	m.Create("s1", "victim", "t1", "")
	m.Create("s2", "victim", "t2", "")
	m.Create("s3", "safe", "t3", "")

	removed := m.ForceLogout("victim")
	if removed != 2 {
		t.Errorf("removed = %d, want 2", removed)
	}
	if m.Count("victim") != 0 {
		t.Error("victim should have 0 sessions")
	}
	if m.Count("safe") != 1 {
		t.Error("safe should still have 1 session")
	}
}

func TestRemove(t *testing.T) {
	m := New(nil)
	m.Create("s1", "c", "t1", "")
	m.Remove("s1")
	if m.Count("c") != 0 {
		t.Error("session should be removed")
	}
}

func TestCleanup(t *testing.T) {
	m := New(nil)
	m.Create("s1", "c", "t1", "")
	// Backdate last_seen
	m.mu.Lock()
	m.sessions["s1"].LastSeen = time.Now().Add(-2 * time.Hour)
	m.mu.Unlock()

	m.Create("s2", "c", "t2", "")

	cleaned := m.Cleanup(1 * time.Hour)
	if cleaned != 1 {
		t.Errorf("cleaned = %d, want 1", cleaned)
	}
	if m.Count("c") != 1 {
		t.Error("should have 1 active session left")
	}
}

func TestTouch(t *testing.T) {
	m := New(nil)
	m.Create("s1", "c", "t1", "")

	before := m.sessions["s1"].LastSeen
	time.Sleep(1 * time.Millisecond)
	m.Touch("s1")
	after := m.sessions["s1"].LastSeen

	if !after.After(before) {
		t.Error("Touch should update LastSeen")
	}
}
