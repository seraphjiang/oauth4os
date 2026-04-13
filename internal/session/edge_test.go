package session

import (
	"sync"
	"testing"
	"time"
)

// Edge: session limit enforced
func TestEdge_SessionLimitEnforced(t *testing.T) {
	m := New(map[string]int{"*": 2})
	m.Create("s1", "app", "t1", "1.2.3.4")
	m.Create("s2", "app", "t2", "1.2.3.4")
	if m.Create("s3", "app", "t3", "1.2.3.4") {
		t.Error("third session should be rejected at limit 2")
	}
}

// Edge: force logout removes all sessions for client
func TestEdge_ForceLogoutAll(t *testing.T) {
	m := New(nil)
	m.Create("s1", "app", "t1", "1.2.3.4")
	m.Create("s2", "app", "t2", "1.2.3.4")
	m.Create("s3", "other", "t3", "1.2.3.4")
	n := m.ForceLogout("app")
	if n != 2 {
		t.Errorf("expected 2 removed, got %d", n)
	}
	if m.Count("app") != 0 {
		t.Error("app should have 0 sessions after force logout")
	}
	if m.Count("other") != 1 {
		t.Error("other client should still have 1 session")
	}
}

// Edge: cleanup removes idle sessions
func TestEdge_CleanupIdle(t *testing.T) {
	m := New(nil)
	m.Create("s1", "app", "t1", "1.2.3.4")
	// Backdate last_seen
	m.mu.Lock()
	m.sessions["s1"].LastSeen = time.Now().Add(-2 * time.Hour)
	m.mu.Unlock()
	n := m.Cleanup(time.Hour)
	if n != 1 {
		t.Errorf("expected 1 cleaned up, got %d", n)
	}
}

// Edge: concurrent create/remove must not panic
func TestEdge_ConcurrentCreateRemove(t *testing.T) {
	m := New(nil)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			m.Create("s"+string(rune(idx)), "app", "t", "1.2.3.4")
		}(i)
		go func(idx int) {
			defer wg.Done()
			m.Remove("s" + string(rune(idx)))
		}(i)
	}
	wg.Wait()
}

// Edge: Touch updates LastSeen
func TestEdge_TouchUpdatesLastSeen(t *testing.T) {
	m := New(nil)
	m.Create("s1", "app", "t1", "1.2.3.4")
	before := m.sessions["s1"].LastSeen
	time.Sleep(time.Millisecond)
	m.Touch("s1")
	after := m.sessions["s1"].LastSeen
	if !after.After(before) {
		t.Error("Touch should update LastSeen")
	}
}

// Edge: List with empty clientID returns all
func TestEdge_ListAll(t *testing.T) {
	m := New(nil)
	m.Create("s1", "a", "t1", "1.2.3.4")
	m.Create("s2", "b", "t2", "1.2.3.4")
	all := m.List("")
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}
}
