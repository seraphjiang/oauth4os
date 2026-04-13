package dpop

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func buildDPoP(typ, alg string, jwk json.RawMessage, htm, htu string, iat int64) string {
	header, _ := json.Marshal(map[string]interface{}{"typ": typ, "alg": alg, "jwk": jwk})
	payload, _ := json.Marshal(map[string]interface{}{"htm": htm, "htu": htu, "iat": iat})
	return base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".fakesig"
}

// Mutation: remove typ check → wrong typ must be rejected
func TestMutation_TypCheck(t *testing.T) {
	tok := buildDPoP("at+jwt", "ES256", json.RawMessage(`{"kty":"EC"}`), "GET", "https://x.com", time.Now().Unix())
	r := httptest.NewRequest("GET", "https://x.com", nil)
	r.Header.Set("DPoP", tok)
	_, err := Validate(r)
	if err == nil {
		t.Error("wrong typ should be rejected")
	}
}

// Mutation: remove JWK nil check → missing JWK must error
func TestMutation_JWKRequired(t *testing.T) {
	header, _ := json.Marshal(map[string]interface{}{"typ": "dpop+jwt", "alg": "ES256"})
	payload, _ := json.Marshal(map[string]interface{}{"htm": "GET", "htu": "https://x.com", "iat": time.Now().Unix()})
	tok := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".sig"
	r := httptest.NewRequest("GET", "https://x.com", nil)
	r.Header.Set("DPoP", tok)
	_, err := Validate(r)
	if err == nil {
		t.Error("missing JWK should be rejected")
	}
}

// Mutation: remove method check → mismatched method must error
func TestMutation_MethodMatch(t *testing.T) {
	tok := buildDPoP("dpop+jwt", "ES256", json.RawMessage(`{"kty":"EC"}`), "POST", "https://x.com", time.Now().Unix())
	r := httptest.NewRequest("GET", "https://x.com", nil)
	r.Header.Set("DPoP", tok)
	_, err := Validate(r)
	if err == nil {
		t.Error("method mismatch should be rejected")
	}
}

// Mutation: remove freshness check → expired proof must error
func TestMutation_Freshness(t *testing.T) {
	old := time.Now().Add(-10 * time.Minute).Unix()
	tok := buildDPoP("dpop+jwt", "ES256", json.RawMessage(`{"kty":"EC"}`), "GET", "https://x.com", old)
	r := httptest.NewRequest("GET", "https://x.com", nil)
	r.Header.Set("DPoP", tok)
	_, err := Validate(r)
	if err == nil {
		t.Error("expired proof should be rejected")
	}
}

// Mutation: remove nil return for empty header → no DPoP must return nil, nil
func TestMutation_EmptyHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "https://x.com", nil)
	proof, err := Validate(r)
	if err != nil {
		t.Errorf("no DPoP header should not error: %v", err)
	}
	if proof != nil {
		t.Error("no DPoP header should return nil proof")
	}
}

// Mutation: change thumbprint hash → different JWKs must produce different thumbprints
func TestMutation_ThumbprintUniqueness(t *testing.T) {
	t1 := JWKThumbprint(json.RawMessage(`{"kty":"EC","crv":"P-256","x":"a","y":"b"}`))
	t2 := JWKThumbprint(json.RawMessage(`{"kty":"EC","crv":"P-256","x":"c","y":"d"}`))
	if t1 == t2 {
		t.Error("different JWKs must produce different thumbprints")
	}
	// Same input → same output
	t3 := JWKThumbprint(json.RawMessage(`{"kty":"EC","crv":"P-256","x":"a","y":"b"}`))
	if t1 != t3 {
		t.Error("same JWK must produce same thumbprint")
	}
}

// Valid proof should succeed
func TestMutation_ValidProof(t *testing.T) {
	tok := buildDPoP("dpop+jwt", "ES256", json.RawMessage(`{"kty":"EC"}`), "GET", "https://x.com", time.Now().Unix())
	r := httptest.NewRequest("GET", "https://x.com", nil)
	r.Header.Set("DPoP", tok)
	proof, err := Validate(r)
	if err != nil {
		t.Fatalf("valid proof should succeed: %v", err)
	}
	if proof.Method != "GET" {
		t.Errorf("method should be GET, got %s", proof.Method)
	}
	if proof.JWKThumbprint == "" {
		t.Error("thumbprint should not be empty")
	}
	fmt.Println("valid proof:", proof.JWKThumbprint[:12]+"...")
}

// Edge: malformed base64 in DPoP header
func TestEdge_MalformedBase64(t *testing.T) {
	r := httptest.NewRequest("GET", "/resource", nil)
	r.Header.Set("DPoP", "!!!.payload.sig")
	_, err := Validate(r)
	if err == nil {
		t.Error("malformed base64 should fail")
	}
}

// Edge: wrong number of JWT parts
func TestEdge_WrongPartCount(t *testing.T) {
	r := httptest.NewRequest("GET", "/resource", nil)
	r.Header.Set("DPoP", "only-two-parts.here")
	_, err := Validate(r)
	if err == nil {
		t.Error("two-part JWT should fail")
	}
}

// Edge: valid JSON but wrong typ
func TestEdge_WrongTyp(t *testing.T) {
	hdr := base64.RawURLEncoding.EncodeToString([]byte(`{"typ":"JWT","alg":"ES256","jwk":{"kty":"EC"}}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"htm":"GET","htu":"/resource","iat":` + fmt.Sprintf("%d", time.Now().Unix()) + `}`))
	r := httptest.NewRequest("GET", "/resource", nil)
	r.Header.Set("DPoP", hdr+"."+payload+".sig")
	_, err := Validate(r)
	if err == nil || !strings.Contains(err.Error(), "typ") {
		t.Errorf("wrong typ should fail with typ error, got: %v", err)
	}
}

// Edge: method mismatch in proof
func TestEdge_MethodMismatch(t *testing.T) {
	hdr := base64.RawURLEncoding.EncodeToString([]byte(`{"typ":"dpop+jwt","alg":"ES256","jwk":{"kty":"EC","crv":"P-256","x":"x","y":"y"}}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"htm":"POST","htu":"/resource","iat":` + fmt.Sprintf("%d", time.Now().Unix()) + `}`))
	r := httptest.NewRequest("GET", "/resource", nil)
	r.Header.Set("DPoP", hdr+"."+payload+".sig")
	_, err := Validate(r)
	if err == nil || !strings.Contains(err.Error(), "method") {
		t.Errorf("method mismatch should fail, got: %v", err)
	}
}
