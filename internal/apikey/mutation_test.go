package apikey

import (
	"testing"
)

// Mutation: skip revocation check — validate always succeeds
func TestMutation_RevokedKeyAccepted(t *testing.T) {
	s := NewStore()
	raw, k := s.Generate("client1", []string{"admin"})
	s.Revoke(k.ID)
	_, valid := s.Validate(raw)
	if valid {
		t.Error("MUTATION SURVIVED: revoked key should be rejected")
	}
}

// Mutation: constant-time compare removed — any prefix matches
func TestMutation_WrongKeyAccepted(t *testing.T) {
	s := NewStore()
	s.Generate("client1", []string{"admin"})
	_, valid := s.Validate("oak_0000000000000000000000000000000000000000000000000000000000000000")
	if valid {
		t.Error("MUTATION SURVIVED: wrong key should be rejected")
	}
}

// Mutation: Generate returns empty key
func TestMutation_EmptyKeyGenerated(t *testing.T) {
	s := NewStore()
	raw, k := s.Generate("client1", []string{"read:logs-*"})
	if raw == "" || k.ID == "" || k.Prefix == "" {
		t.Error("MUTATION SURVIVED: generate returned empty values")
	}
	if len(raw) < 20 {
		t.Error("MUTATION SURVIVED: key too short")
	}
}

// Mutation: scopes not stored
func TestMutation_ScopesLost(t *testing.T) {
	s := NewStore()
	raw, _ := s.Generate("client1", []string{"read:logs-*", "admin"})
	claims, valid := s.Validate(raw)
	if !valid {
		t.Fatal("key should be valid")
	}
	if len(claims.Scopes) != 2 {
		t.Errorf("MUTATION SURVIVED: expected 2 scopes, got %d", len(claims.Scopes))
	}
}

// Mutation: List returns all clients' keys
func TestMutation_ListLeaksOtherClients(t *testing.T) {
	s := NewStore()
	s.Generate("client1", []string{"admin"})
	s.Generate("client2", []string{"read:logs-*"})
	list := s.List("client1")
	for _, k := range list {
		if k.ClientID != "client1" {
			t.Error("MUTATION SURVIVED: List returned another client's key")
		}
	}
}

// Mutation: Revoke returns true for nonexistent key
func TestMutation_RevokeNonexistent(t *testing.T) {
	s := NewStore()
	if s.Revoke("nonexistent") {
		t.Error("MUTATION SURVIVED: revoke should return false for unknown key")
	}
}
