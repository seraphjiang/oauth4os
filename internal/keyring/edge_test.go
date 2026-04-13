package keyring

import (
	"testing"
	"time"
)

// Edge: Current returns non-nil key
func TestEdge_CurrentNonNil(t *testing.T) {
	r, err := New(2048, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	k := r.Current()
	if k == nil {
		t.Error("Current should return non-nil key")
	}
}

// Edge: JWKS returns valid JSON
func TestEdge_JWKSValid(t *testing.T) {
	r, err := New(2048, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	jwks := r.JWKS()
	if len(jwks) == 0 {
		t.Error("JWKS should return non-empty bytes")
	}
}

// Edge: Current key has KID
func TestEdge_CurrentHasKID(t *testing.T) {
	r, err := New(2048, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	k := r.Current()
	if k.KID == "" {
		t.Error("key should have non-empty KID")
	}
}
