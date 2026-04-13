package apikey

import (
	"testing"
	"time"
)

func TestEdge_GeneratedKeyHasCreatedAt(t *testing.T) {
	s := NewStore()
	_, k := s.Generate("app", []string{"read"})
	if k.CreatedAt.IsZero() || k.CreatedAt.After(time.Now().Add(time.Second)) {
		t.Error("CreatedAt should be set to approximately now")
	}
}

func TestEdge_GeneratedKeyHasScopes(t *testing.T) {
	s := NewStore()
	_, k := s.Generate("app", []string{"read", "write"})
	if len(k.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(k.Scopes))
	}
}

func TestEdge_ListAllClients(t *testing.T) {
	s := NewStore()
	s.Generate("a", []string{"read"})
	s.Generate("b", []string{"write"})
	a := s.List("a")
	b := s.List("b")
	if len(a) != 1 || len(b) != 1 {
		t.Errorf("expected 1 each, got a=%d b=%d", len(a), len(b))
	}
}
