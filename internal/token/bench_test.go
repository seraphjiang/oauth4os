package token

import "testing"

func BenchmarkCreateToken(b *testing.B) {
	m := NewManager()
	m.RegisterClient("bench-client", "secret", []string{"read:logs-*"}, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.CreateTokenForClient("bench-client", []string{"read:logs-*"})
	}
}

func BenchmarkLookup(b *testing.B) {
	m := NewManager()
	m.RegisterClient("bench-client", "secret", []string{"read:logs-*"}, nil)
	tok, _ := m.CreateTokenForClient("bench-client", []string{"read:logs-*"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Lookup(tok.ID)
	}
}

func BenchmarkIsValid(b *testing.B) {
	m := NewManager()
	m.RegisterClient("bench-client", "secret", []string{"read:logs-*"}, nil)
	tok, _ := m.CreateTokenForClient("bench-client", []string{"read:logs-*"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.IsValid(tok.ID)
	}
}

func BenchmarkTouchToken(b *testing.B) {
	m := NewManager()
	m.RegisterClient("bench-client", "secret", []string{"read:logs-*"}, nil)
	tok, _ := m.CreateTokenForClient("bench-client", []string{"read:logs-*"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.TouchToken(tok.ID, 0)
	}
}
