package session

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSessionLimit_Enforced(t *testing.T) {
	m := New(map[string]int{"c": 2})
	if !m.Create("s1", "c", "t1", "1.2.3.4") {
		t.Error("first session should succeed")
	}
	if !m.Create("s2", "c", "t2", "1.2.3.4") {
		t.Error("second session should succeed")
	}
	if m.Create("s3", "c", "t3", "1.2.3.4") {
		t.Error("third session should be rejected (limit=2)")
	}
}

func TestSessionLimit_PerClient(t *testing.T) {
	m := New(map[string]int{"a": 1, "b": 3})
	m.Create("s1", "a", "t1", "ip")
	if m.Create("s2", "a", "t2", "ip") {
		t.Error("client a should be limited to 1")
	}
	for i := 0; i < 3; i++ {
		if !m.Create(fmt.Sprintf("b%d", i), "b", "t", "ip") {
			t.Errorf("client b session %d should succeed", i)
		}
	}
}

func TestSessionLimit_GlobalDefault(t *testing.T) {
	m := New(map[string]int{"*": 1})
	m.Create("s1", "unknown-client", "t1", "ip")
	if m.Create("s2", "unknown-client", "t2", "ip") {
		t.Error("global default limit should apply")
	}
}

func TestForceLogout_Edge(t *testing.T) {
	m := New(nil)
	m.Create("s1", "c", "t1", "ip")
	m.Create("s2", "c", "t2", "ip")
	m.Create("s3", "other", "t3", "ip")

	n := m.ForceLogout("c")
	if n != 2 {
		t.Errorf("expected 2 removed, got %d", n)
	}
	if m.Count("c") != 0 {
		t.Error("client c should have 0 sessions")
	}
	if m.Count("other") != 1 {
		t.Error("other client should be unaffected")
	}
}

func TestCleanup_IdleSessions(t *testing.T) {
	m := New(nil)
	m.Create("old", "c", "t1", "ip")
	m.Create("fresh", "c", "t2", "ip")

	// Manually age the old session
	m.mu.Lock()
	m.sessions["old"].LastSeen = time.Now().Add(-2 * time.Hour)
	m.mu.Unlock()

	n := m.Cleanup(1 * time.Hour)
	if n != 1 {
		t.Errorf("expected 1 cleaned, got %d", n)
	}
	if m.Count("c") != 1 {
		t.Error("fresh session should survive")
	}
}

func TestTouch_UpdatesLastSeen(t *testing.T) {
	m := New(nil)
	m.Create("s1", "c", "t1", "ip")

	m.mu.RLock()
	before := m.sessions["s1"].LastSeen
	m.mu.RUnlock()

	time.Sleep(1 * time.Millisecond)
	m.Touch("s1")

	m.mu.RLock()
	after := m.sessions["s1"].LastSeen
	m.mu.RUnlock()

	if !after.After(before) {
		t.Error("Touch should update LastSeen")
	}
}

func TestList_AllClients(t *testing.T) {
	m := New(nil)
	m.Create("s1", "a", "t1", "ip")
	m.Create("s2", "b", "t2", "ip")
	all := m.List("")
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}
}

func TestRemove_NonexistentSession(t *testing.T) {
	m := New(nil)
	m.Remove("nonexistent") // should not panic
}

func TestConcurrentSessionOps(t *testing.T) {
	m := New(map[string]int{"*": 1000})
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("s%d", n)
			m.Create(id, "c", "t", "ip")
			m.Touch(id)
			m.List("c")
			m.Count("c")
		}(i)
	}
	wg.Wait()

	// Force logout all
	m.ForceLogout("c")
	if m.Count("c") != 0 {
		t.Error("all sessions should be removed")
	}
}
