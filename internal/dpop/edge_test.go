package dpop

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// Edge: no DPoP header returns nil proof (not error)
func TestEdge_NoDPoPReturnsNil(t *testing.T) {
	r := httptest.NewRequest("GET", "/resource", nil)
	proof, err := Validate(r)
	if err != nil {
		t.Errorf("no DPoP should not error: %v", err)
	}
	if proof != nil {
		t.Error("no DPoP should return nil proof")
	}
}

// Edge: JWKThumbprint produces consistent output
func TestEdge_ThumbprintConsistent(t *testing.T) {
	jwk := json.RawMessage(`{"kty":"EC","crv":"P-256","x":"test","y":"test"}`)
	t1 := JWKThumbprint(jwk)
	t2 := JWKThumbprint(jwk)
	if t1 != t2 {
		t.Error("same JWK should produce same thumbprint")
	}
	if t1 == "" {
		t.Error("thumbprint should not be empty")
	}
}

// Edge: different JWKs produce different thumbprints
func TestEdge_DifferentJWKsDifferentThumbprints(t *testing.T) {
	jwk1 := json.RawMessage(`{"kty":"EC","crv":"P-256","x":"a","y":"a"}`)
	jwk2 := json.RawMessage(`{"kty":"EC","crv":"P-256","x":"b","y":"b"}`)
	if JWKThumbprint(jwk1) == JWKThumbprint(jwk2) {
		t.Error("different JWKs should produce different thumbprints")
	}
}
