package dpop

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMissingJWK(t *testing.T) {
	header, _ := json.Marshal(map[string]interface{}{"typ": "dpop+jwt", "alg": "RS256"})
	payload, _ := json.Marshal(map[string]interface{}{"htm": "GET", "htu": "/", "iat": time.Now().Unix()})
	token := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("DPoP", token)
	_, err := Validate(r)
	if err == nil {
		t.Fatal("expected error for missing jwk")
	}
}

func TestMalformedHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("DPoP", "not.valid.jwt")
	_, err := Validate(r)
	if err == nil {
		t.Fatal("expected error for malformed header")
	}
}

func TestTwoPartToken(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("DPoP", "only.two")
	_, err := Validate(r)
	if err == nil {
		t.Fatal("expected error for 2-part token")
	}
}

func TestThumbprintDeterministic(t *testing.T) {
	jwk1 := json.RawMessage(`{"kty":"RSA","n":"abc","e":"AQAB"}`)
	jwk2 := json.RawMessage(`{"kty":"RSA","n":"abc","e":"AQAB"}`)
	if JWKThumbprint(jwk1) != JWKThumbprint(jwk2) {
		t.Fatal("same JWK should produce same thumbprint")
	}
}

func TestFreshProof(t *testing.T) {
	jwk := json.RawMessage(`{"kty":"RSA","n":"test","e":"AQAB"}`)
	token := makeDPoP("dpop+jwt", "RS256", jwk, "GET", "/resource", time.Now().Unix())
	r := httptest.NewRequest("GET", "/resource", nil)
	r.Header.Set("DPoP", token)
	proof, err := Validate(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if time.Since(proof.IssuedAt) > 10*time.Second {
		t.Fatal("proof should be fresh")
	}
}
