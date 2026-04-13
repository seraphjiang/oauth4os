package token

import (
	"sync"
	"testing"
)

func TestEdge_ConcurrentRegisterClient(t *testing.T) {
	m := NewManager()
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			m.RegisterClient(string(rune('a'+n%26)), "s", []string{"read"}, nil)
		}(i)
	}
	wg.Wait()
}

func TestEdge_ConcurrentIsValid(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "s", []string{"read"}, nil)
	tok, _ := m.CreateTokenForClient("app", []string{"read"})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.IsValid(tok.ID)
		}()
	}
	wg.Wait()
}

func TestEdge_RevokeByClientEmpty(t *testing.T) {
	m := NewManager()
	n := m.RevokeByClient("nonexistent")
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}
