package token

import (
	"testing"
)

func TestEdge_AuthenticateClientValid(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "secret", []string{"read"}, nil)
	if err := m.AuthenticateClient("app", "secret"); err != nil {
		t.Errorf("valid credentials should pass: %v", err)
	}
}

func TestEdge_AuthenticateClientWrongSecret(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "secret", []string{"read"}, nil)
	if err := m.AuthenticateClient("app", "wrong"); err == nil {
		t.Error("wrong secret should fail")
	}
}

func TestEdge_AuthenticateClientUnknown(t *testing.T) {
	m := NewManager()
	if err := m.AuthenticateClient("unknown", "secret"); err == nil {
		t.Error("unknown client should fail")
	}
}

func TestEdge_ListClientsEmpty(t *testing.T) {
	m := NewManager()
	c := m.ListClients()
	if len(c) != 0 {
		t.Errorf("empty manager should list 0 clients, got %d", len(c))
	}
}

func TestEdge_ListClientsMultiple(t *testing.T) {
	m := NewManager()
	m.RegisterClient("a", "s1", []string{"read"}, nil)
	m.RegisterClient("b", "s2", []string{"write"}, nil)
	c := m.ListClients()
	if len(c) != 2 {
		t.Errorf("expected 2 clients, got %d", len(c))
	}
}

func TestEdge_BindDPoPUnknownToken(t *testing.T) {
	m := NewManager()
	m.BindDPoP("nonexistent", "thumbprint")
	// Should not panic
}

func TestEdge_VerifyDPoPUnbound(t *testing.T) {
	m := NewManager()
	if m.VerifyDPoP("nonexistent", "thumbprint") {
		t.Error("unbound token should not verify DPoP")
	}
}
