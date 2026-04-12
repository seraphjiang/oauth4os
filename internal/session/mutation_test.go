package session

import (
	"testing"
	"time"
)

// Mutation: remove limit check → Create must enforce session limits
func TestMutation_SessionLimit(t *testing.T) {
	m := New(map[string]int{"app": 2})
	m.Create("s1", "app", "t1", "1.2.3.4")
	m.Create("s2", "app", "t2", "1.2.3.4")
	ok := m.Create("s3", "app", "t3", "1.2.3.4")
	if ok {
		t.Error("third session should be rejected (limit=2)")
	}
}

// Mutation: remove session from map in Remove → Remove must delete session
func TestMutation_RemoveDeletes(t *testing.T) {
	m := New(nil)
	m.Create("s1", "app", "t1", "1.2.3.4")
	m.Remove("s1")
	if m.Count("app") != 0 {
		t.Error("Remove must delete the session")
	}
}

// Mutation: remove ForceLogout → must revoke all sessions for client
func TestMutation_ForceLogout(t *testing.T) {
	m := New(nil)
	m.Create("s1", "app", "t1", "1.2.3.4")
	m.Create("s2", "app", "t2", "1.2.3.4")
	m.Create("s3", "other", "t3", "1.2.3.4")
	n := m.ForceLogout("app")
	if n != 2 {
		t.Errorf("ForceLogout should remove 2 sessions, removed %d", n)
	}
	if m.Count("other") != 1 {
		t.Error("ForceLogout must not affect other clients")
	}
}

// Mutation: remove Touch update → Touch must update LastActive
func TestMutation_TouchUpdates(t *testing.T) {
	m := New(nil)
	m.Create("s1", "app", "t1", "1.2.3.4")
	time.Sleep(10 * time.Millisecond)
	m.Touch("s1")
	sessions := m.List("app")
	if len(sessions) != 1 {
		t.Fatal("expected 1 session")
	}
	if time.Since(sessions[0].LastSeen) > 50*time.Millisecond {
		t.Error("Touch must update LastActive")
	}
}

// Mutation: remove Cleanup → must remove idle sessions
func TestMutation_Cleanup(t *testing.T) {
	m := New(nil)
	m.Create("s1", "app", "t1", "1.2.3.4")
	time.Sleep(50 * time.Millisecond)
	n := m.Cleanup(10 * time.Millisecond)
	if n != 1 {
		t.Errorf("Cleanup should remove 1 idle session, removed %d", n)
	}
}

// Mutation: remove List → must return active sessions
func TestMutation_ListSessions(t *testing.T) {
	m := New(nil)
	m.Create("s1", "app", "t1", "1.2.3.4")
	m.Create("s2", "app", "t2", "5.6.7.8")
	list := m.List("app")
	if len(list) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(list))
	}
}

// Mutation: remove Count → must reflect active session count
func TestMutation_CountAccurate(t *testing.T) {
	m := New(nil)
	m.Create("s1", "app", "t1", "1.2.3.4")
	m.Create("s2", "app", "t2", "5.6.7.8")
	m.Create("s3", "other", "t3", "9.0.0.1")
	if m.Count("app") != 2 {
		t.Errorf("expected 2 for app, got %d", m.Count("app"))
	}
	if m.Count("other") != 1 {
		t.Errorf("expected 1 for other, got %d", m.Count("other"))
	}
}
