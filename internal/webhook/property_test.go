package webhook

import (
	"crypto/rand"
	"testing"
)

// Property: Sign+Verify round-trip must always succeed for any payload
func TestProperty_SignVerifyRoundTrip(t *testing.T) {
	s := NewSender("test-secret-key")
	for i := 0; i < 100; i++ {
		payload := make([]byte, i*10)
		rand.Read(payload)
		sig := s.Sign(payload)
		if !s.Verify(payload, sig) {
			t.Fatalf("round-trip failed for payload size %d", len(payload))
		}
	}
}

// Property: different payloads must produce different signatures
func TestProperty_UniqueSignatures(t *testing.T) {
	s := NewSender("secret")
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		payload := make([]byte, 32)
		rand.Read(payload)
		sig := s.Sign(payload)
		if seen[sig] {
			t.Fatal("collision: two different payloads produced same signature")
		}
		seen[sig] = true
	}
}
