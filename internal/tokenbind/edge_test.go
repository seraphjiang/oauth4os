package tokenbind

import "testing"

// Edge: Remove non-existent binding is no-op
func TestEdge_RemoveNonExistent(t *testing.T) {
	b := New()
	b.Remove("nonexistent") // should not panic
}

// Edge: Bind and verify
func TestEdge_BindAndCheck(t *testing.T) {
	b := New()
	b.Bind("tok_123", "fingerprint_abc")
	if !b.Verify("tok_123", "fingerprint_abc") {
		t.Error("bound token should verify")
	}
	if b.Verify("tok_123", "wrong") {
		t.Error("wrong fingerprint should fail")
	}
}
