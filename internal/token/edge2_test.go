package token

import (
	"sync"
	"testing"
)

func TestEdge_ConcurrentLookup(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "s", []string{"read"}, nil)
	tok, _ := m.CreateTokenForClient("app", []string{"read"})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Lookup(tok.ID)
		}()
	}
	wg.Wait()
}

func TestEdge_LookupNonexistent(t *testing.T) {
	m := NewManager()
	_, _, _, _, _, ok := m.Lookup("nonexistent")
	if ok {
		t.Error("nonexistent token should return ok=false")
	}
}

func TestEdge_RegisterDuplicateClient(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "s1", []string{"read"}, nil)
	m.RegisterClient("app", "s2", []string{"write"}, nil)
	// Should not panic — last registration wins or merges
	clients := m.Clients()
	found := false
	for _, c := range clients {
		if c.ID == "app" {
			found = true
		}
	}
	if !found {
		t.Error("client should exist after registration")
	}
}

func TestEdge_ValidateRedirectURIUnregistered(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "s", []string{"read"}, []string{"http://localhost/cb"})
	if m.ValidateRedirectURI("app", "http://evil.com/cb") {
		t.Error("unregistered redirect URI should be rejected")
	}
}

func TestEdge_ValidateRedirectURIRegistered(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "s", []string{"read"}, []string{"http://localhost/cb"})
	if !m.ValidateRedirectURI("app", "http://localhost/cb") {
		t.Error("registered redirect URI should be accepted")
	}
}
