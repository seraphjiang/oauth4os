package tokenbind

import (
	"net/http/httptest"
	"testing"
)

// Mutation: remove first-use-only guard → Bind must not overwrite existing binding
func TestMutation_BindOnce(t *testing.T) {
	b := New()
	b.Bind("tok_1", "fp_original")
	b.Bind("tok_1", "fp_attacker")
	if !b.Verify("tok_1", "fp_original") {
		t.Error("original fingerprint must still be valid")
	}
	if b.Verify("tok_1", "fp_attacker") {
		t.Error("attacker fingerprint must be rejected")
	}
}

// Mutation: remove fingerprint comparison → mismatched fingerprint must fail
func TestMutation_VerifyMismatch(t *testing.T) {
	b := New()
	b.Bind("tok_2", "fp_a")
	if b.Verify("tok_2", "fp_b") {
		t.Error("mismatched fingerprint must fail verification")
	}
}

// Mutation: remove unbound=allow → unbound tokens must be allowed
func TestMutation_UnboundAllowed(t *testing.T) {
	b := New()
	if !b.Verify("tok_unknown", "any_fp") {
		t.Error("unbound token must be allowed")
	}
}

// Mutation: remove Remove → Remove must delete binding
func TestMutation_RemoveBinding(t *testing.T) {
	b := New()
	b.Bind("tok_3", "fp_x")
	b.Remove("tok_3")
	// After removal, any fingerprint should work (unbound)
	if !b.Verify("tok_3", "fp_different") {
		t.Error("removed binding should allow any fingerprint")
	}
}

// Mutation: remove IP from fingerprint → different IPs must produce different fingerprints
func TestMutation_FingerprintIncludesIP(t *testing.T) {
	r1 := httptest.NewRequest("GET", "/", nil)
	r1.RemoteAddr = "1.2.3.4:1234"
	r1.Header.Set("User-Agent", "same-agent")

	r2 := httptest.NewRequest("GET", "/", nil)
	r2.RemoteAddr = "5.6.7.8:1234"
	r2.Header.Set("User-Agent", "same-agent")

	if Fingerprint(r1) == Fingerprint(r2) {
		t.Error("different IPs must produce different fingerprints")
	}
}
