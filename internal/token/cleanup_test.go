package token

import (
	"testing"
	"time"
)

func TestCleanupExpired(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)

	tok1, _ := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})
	tok2, _ := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})

	// Expire tok1
	m.mu.Lock()
	m.tokens[tok1.ID].ExpiresAt = time.Now().Add(-1 * time.Second)
	m.mu.Unlock()

	removed := m.Cleanup()
	if removed != 1 {
		t.Fatalf("expected 1 removed, got %d", removed)
	}
	if m.IsValid(tok1.ID) {
		t.Fatal("expired token should be cleaned up")
	}
	if !m.IsValid(tok2.ID) {
		t.Fatal("valid token should remain")
	}
}

func TestCleanupRevoked(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)

	tok, _ := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})
	m.mu.Lock()
	m.tokens[tok.ID].Revoked = true
	m.mu.Unlock()

	removed := m.Cleanup()
	if removed != 1 {
		t.Fatalf("expected 1 removed, got %d", removed)
	}
}

func TestCleanupEmpty(t *testing.T) {
	m := NewManager()
	removed := m.Cleanup()
	if removed != 0 {
		t.Fatalf("expected 0 removed, got %d", removed)
	}
}

func BenchmarkCleanup(b *testing.B) {
	m := NewManager()
	m.RegisterClient("bench", "secret", []string{"read:logs-*"}, nil)
	// Pre-populate with 1000 expired tokens
	for i := 0; i < 1000; i++ {
		tok, _ := m.CreateTokenForClient("bench", []string{"read:logs-*"})
		m.mu.Lock()
		m.tokens[tok.ID].ExpiresAt = time.Now().Add(-1 * time.Second)
		m.mu.Unlock()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Cleanup()
		// Re-populate for next iteration
		for j := 0; j < 1000; j++ {
			tok, _ := m.CreateTokenForClient("bench", []string{"read:logs-*"})
			m.mu.Lock()
			m.tokens[tok.ID].ExpiresAt = time.Now().Add(-1 * time.Second)
			m.mu.Unlock()
		}
	}
}
