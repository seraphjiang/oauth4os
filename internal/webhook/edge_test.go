package webhook

import "testing"

// Edge: NewSender with empty secret
func TestEdge_EmptySecretSender(t *testing.T) {
	s := NewSender("")
	sig := s.Sign([]byte("test"))
	if sig == "" {
		t.Error("even empty secret should produce signature")
	}
}

// Edge: Verify with empty body
func TestEdge_VerifyEmptyBody(t *testing.T) {
	s := NewSender("secret")
	sig := s.Sign([]byte{})
	if !s.Verify([]byte{}, sig) {
		t.Error("empty body sign+verify should match")
	}
}
