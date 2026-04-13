package session

import (
	"fmt"
	"sync"
	"testing"
)

// Property: concurrent Create+List+Remove never corrupts state
func TestProperty_ConcurrentOps(t *testing.T) {
	m := New(nil)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		id := fmt.Sprintf("s%d", i)
		go func() { defer wg.Done(); m.Create(id, "app", "t", "1.2.3.4") }()
		go func() { defer wg.Done(); m.List("app") }()
		go func() { defer wg.Done(); m.Remove(id) }()
	}
	wg.Wait()
	// State must be consistent — Count must not be negative
	if c := m.Count("app"); c < 0 {
		t.Errorf("count should not be negative, got %d", c)
	}
}

// Property: ForceLogout + Create concurrent must not panic
func TestProperty_ForceLogoutConcurrent(t *testing.T) {
	m := New(nil)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			m.Create(fmt.Sprintf("s%d", idx), "app", "t", "1.2.3.4")
		}(i)
		go func() {
			defer wg.Done()
			m.ForceLogout("app")
		}()
	}
	wg.Wait()
}
