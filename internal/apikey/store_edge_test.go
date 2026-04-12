package apikey

import (
	"sync"
	"testing"
)

func TestConcurrentGenerateAndValidate(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup
	keys := make([]string, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			raw, _ := s.Generate("svc-a", []string{"read:logs-*"})
			keys[n] = raw
		}(i)
	}
	wg.Wait()

	// All keys should be valid
	for i, k := range keys {
		if _, ok := s.Validate(k); !ok {
			t.Fatalf("key %d should be valid", i)
		}
	}
}

func TestRevokeByID(t *testing.T) {
	s := NewStore()
	raw1, k1 := s.Generate("svc-a", []string{"read:logs-*"})
	raw2, _ := s.Generate("svc-a", []string{"admin"})

	s.Revoke(k1.ID)

	if _, ok := s.Validate(raw1); ok {
		t.Fatal("revoked key should fail")
	}
	if _, ok := s.Validate(raw2); !ok {
		t.Fatal("other key should still work")
	}
}

func TestRevokeNonexistent(t *testing.T) {
	s := NewStore()
	if s.Revoke("nonexistent") {
		t.Fatal("should return false for unknown ID")
	}
}

func TestListExcludesRevoked(t *testing.T) {
	s := NewStore()
	_, k1 := s.Generate("svc-a", []string{"read:logs-*"})
	s.Generate("svc-a", []string{"admin"})
	s.Revoke(k1.ID)

	keys := s.List("svc-a")
	if len(keys) != 1 {
		t.Fatalf("expected 1 active key, got %d", len(keys))
	}
}

func TestLastUsedUpdated(t *testing.T) {
	s := NewStore()
	raw, _ := s.Generate("svc-a", []string{"read:logs-*"})

	claims, _ := s.Validate(raw)
	if claims.KeyID == "" {
		t.Fatal("expected key ID in claims")
	}

	// Validate again — LastUsed should be set
	s.mu.RLock()
	k := s.keys[raw]
	s.mu.RUnlock()
	if k.LastUsed.IsZero() {
		t.Fatal("LastUsed should be set after Validate")
	}
}

func TestKeyPrefix(t *testing.T) {
	s := NewStore()
	raw, k := s.Generate("svc-a", nil)
	if raw[:4] != "oak_" {
		t.Fatalf("expected oak_ prefix, got %s", raw[:4])
	}
	if k.Prefix != raw[:12] {
		t.Fatalf("prefix mismatch: %s vs %s", k.Prefix, raw[:12])
	}
}
