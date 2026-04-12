package tokenbind

import (
	"net/http/httptest"
	"testing"
)

func TestBindAndVerify(t *testing.T) {
	b := New()
	b.Bind("tok123", "fp-abc")
	if !b.Verify("tok123", "fp-abc") {
		t.Fatal("expected match")
	}
	if b.Verify("tok123", "fp-different") {
		t.Fatal("expected mismatch")
	}
}

func TestUnboundAllowed(t *testing.T) {
	b := New()
	if !b.Verify("unknown", "any-fp") {
		t.Fatal("unbound tokens should be allowed")
	}
}

func TestRemove(t *testing.T) {
	b := New()
	b.Bind("tok123", "fp-abc")
	b.Remove("tok123")
	if !b.Verify("tok123", "fp-different") {
		t.Fatal("removed binding should allow any fingerprint")
	}
}

func TestFingerprint(t *testing.T) {
	r1 := httptest.NewRequest("GET", "/", nil)
	r1.RemoteAddr = "10.0.0.1:1234"
	r1.Header.Set("User-Agent", "cli/1.0")
	fp1 := Fingerprint(r1)

	r2 := httptest.NewRequest("GET", "/", nil)
	r2.RemoteAddr = "10.0.0.1:1234"
	r2.Header.Set("User-Agent", "cli/1.0")
	fp2 := Fingerprint(r2)

	if fp1 != fp2 {
		t.Fatal("same client should produce same fingerprint")
	}

	r3 := httptest.NewRequest("GET", "/", nil)
	r3.RemoteAddr = "10.0.0.2:5678"
	r3.Header.Set("User-Agent", "browser/2.0")
	fp3 := Fingerprint(r3)

	if fp1 == fp3 {
		t.Fatal("different clients should produce different fingerprints")
	}
}
