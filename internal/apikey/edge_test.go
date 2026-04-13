package apikey

import "testing"

// Edge: Generate returns non-empty key
func TestEdge_GenerateNonEmpty(t *testing.T) {
	s := NewStore()
	raw, k := s.Generate("app", []string{"read"})
	if raw == "" || k == nil {
		t.Error("Generate should return non-empty key")
	}
}

// Edge: Validate accepts valid key
func TestEdge_ValidateAccepts(t *testing.T) {
	s := NewStore()
	raw, _ := s.Generate("app", []string{"read"})
	claims, ok := s.Validate(raw)
	if !ok || claims == nil {
		t.Error("Validate should accept valid key")
	}
	if claims.ClientID != "app" {
		t.Errorf("expected client 'app', got %q", claims.ClientID)
	}
}

// Edge: Validate rejects invalid key
func TestEdge_ValidateRejectsInvalid(t *testing.T) {
	s := NewStore()
	_, ok := s.Validate("invalid-key-12345")
	if ok {
		t.Error("Validate should reject invalid key")
	}
}

// Edge: Revoke makes key invalid
func TestEdge_RevokeInvalidates(t *testing.T) {
	s := NewStore()
	raw, k := s.Generate("app", []string{"read"})
	s.Revoke(k.ID)
	_, ok := s.Validate(raw)
	if ok {
		t.Error("revoked key should be invalid")
	}
}

// Edge: List returns keys for specific client
func TestEdge_ListByClient(t *testing.T) {
	s := NewStore()
	s.Generate("app-a", []string{"read"})
	s.Generate("app-b", []string{"write"})
	keys := s.List("app-a")
	if len(keys) != 1 {
		t.Errorf("expected 1 key for app-a, got %d", len(keys))
	}
}

// Edge: unique key IDs
func TestEdge_UniqueIDs(t *testing.T) {
	s := NewStore()
	_, k1 := s.Generate("app", []string{"read"})
	_, k2 := s.Generate("app", []string{"read"})
	if k1.ID == k2.ID {
		t.Error("key IDs should be unique")
	}
}
