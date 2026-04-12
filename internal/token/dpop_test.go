package token

import "testing"

func TestDPoPBindAndVerify(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)
	tok, _ := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})

	m.BindDPoP(tok.ID, "thumbprint-abc123")

	if !m.VerifyDPoP(tok.ID, "thumbprint-abc123") {
		t.Fatal("matching thumbprint should pass")
	}
	if m.VerifyDPoP(tok.ID, "wrong-thumbprint") {
		t.Fatal("wrong thumbprint should fail")
	}
}

func TestDPoPUnboundPasses(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)
	tok, _ := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})

	// No DPoP binding — any thumbprint should pass
	if !m.VerifyDPoP(tok.ID, "any-thumbprint") {
		t.Fatal("unbound token should pass any thumbprint")
	}
}

func TestDPoPNonexistentToken(t *testing.T) {
	m := NewManager()
	if m.VerifyDPoP("nonexistent", "thumb") {
		t.Fatal("nonexistent token should fail")
	}
}
