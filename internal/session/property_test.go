package session

import (
	"strconv"
	"sync"
	"testing"
)

// Property: concurrent Create/Remove must not corrupt state
func TestProperty_ConcurrentCreateRemove(t *testing.T) {
	m := New(nil)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "s-" + strconv.Itoa(n)
			m.Create(id, "app", "t-"+strconv.Itoa(n), "1.2.3.4")
			m.Touch(id)
			m.Remove(id)
		}(i)
	}
	wg.Wait()
	if m.Count("app") != 0 {
		t.Errorf("all sessions removed, count should be 0, got %d", m.Count("app"))
	}
}

// Property: ForceLogout under concurrent access must not panic
func TestProperty_ConcurrentForceLogout(t *testing.T) {
	m := New(nil)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			m.Create("s-"+strconv.Itoa(n), "app", "t", "1.2.3.4")
		}(i)
		go func() {
			defer wg.Done()
			m.ForceLogout("app")
		}()
	}
	wg.Wait()
}
