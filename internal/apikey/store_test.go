package apikey

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateAndValidate(t *testing.T) {
	s := NewStore()
	raw, k := s.Generate("svc-a", []string{"read:logs-*"})
	if k.Prefix != raw[:12] {
		t.Fatalf("prefix mismatch")
	}
	claims, ok := s.Validate(raw)
	if !ok {
		t.Fatal("expected valid")
	}
	if claims.ClientID != "svc-a" {
		t.Fatalf("expected svc-a, got %s", claims.ClientID)
	}
}

func TestInvalidKey(t *testing.T) {
	s := NewStore()
	_, ok := s.Validate("oak_bogus")
	if ok {
		t.Fatal("expected invalid")
	}
}

func TestRevoke(t *testing.T) {
	s := NewStore()
	raw, k := s.Generate("svc-a", []string{"read:logs-*"})
	s.Revoke(k.ID)
	_, ok := s.Validate(raw)
	if ok {
		t.Fatal("expected revoked key to fail")
	}
}

func TestList(t *testing.T) {
	s := NewStore()
	s.Generate("svc-a", []string{"read:logs-*"})
	s.Generate("svc-a", []string{"admin"})
	s.Generate("svc-b", []string{"read:logs-*"})
	keys := s.List("svc-a")
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys for svc-a, got %d", len(keys))
	}
}

func TestExtractKey(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-API-Key", "oak_test123")
	if ExtractKey(r) != "oak_test123" {
		t.Fatal("extract failed")
	}
}
