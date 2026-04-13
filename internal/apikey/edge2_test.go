package apikey

import (
	"sync"
	"testing"
)

func TestEdge_ConcurrentGenerateValidate(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup
	keys := make(chan string, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			raw, _ := s.Generate("app", []string{"read"})
			keys <- raw
		}()
	}
	wg.Wait()
	close(keys)
	for raw := range keys {
		if _, ok := s.Validate(raw); !ok {
			t.Error("generated key should validate")
		}
	}
}

func TestEdge_RevokeNonexistent(t *testing.T) {
	s := NewStore()
	ok := s.Revoke("nonexistent-id")
	if ok {
		t.Error("revoking nonexistent key should return false")
	}
}

func TestEdge_ListEmpty(t *testing.T) {
	s := NewStore()
	keys := s.List("no-client")
	if len(keys) != 0 {
		t.Errorf("empty list should return 0, got %d", len(keys))
	}
}
