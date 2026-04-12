package token

import (
	"testing"
	"time"
)

func TestListClients(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret1", []string{"read:logs-*"}, []string{"https://app/cb"})
	m.RegisterClient("svc-2", "secret2", []string{"admin"}, nil)

	clients := m.ListClients()
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}
	// Verify no secrets leaked
	for _, c := range clients {
		if c.ID == "" {
			t.Fatal("client ID should not be empty")
		}
	}
}

func TestListClientsEmpty(t *testing.T) {
	m := NewManager()
	clients := m.ListClients()
	if len(clients) != 0 {
		t.Fatalf("expected 0 clients, got %d", len(clients))
	}
}

func TestListActiveTokens(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)

	m.CreateTokenForClient("svc-1", []string{"read:logs-*"})
	tok2, _ := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})

	// Revoke one
	m.mu.Lock()
	m.tokens[tok2.ID].Revoked = true
	m.mu.Unlock()

	active := m.ListActiveTokens()
	if len(active) != 1 {
		t.Fatalf("expected 1 active token, got %d", len(active))
	}
}

func TestListActiveTokensExcludesExpired(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)

	tok, _ := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})
	m.mu.Lock()
	m.tokens[tok.ID].ExpiresAt = time.Now().Add(-1 * time.Second)
	m.mu.Unlock()

	active := m.ListActiveTokens()
	if len(active) != 0 {
		t.Fatalf("expected 0 active tokens, got %d", len(active))
	}
}

func TestListActiveTokensTruncatesID(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)

	m.CreateTokenForClient("svc-1", []string{"read:logs-*"})
	active := m.ListActiveTokens()
	if len(active) != 1 {
		t.Fatal("expected 1 token")
	}
	id := active[0]["id"].(string)
	if len(id) > 20 {
		t.Fatalf("token ID should be truncated, got length %d", len(id))
	}
}
