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

func makeDPoP(typ, alg string, jwk json.RawMessage, htm, htu string, iat int64) string {
	header, _ := json.Marshal(map[string]interface{}{"typ": typ, "alg": alg, "jwk": jwk})
	payload, _ := json.Marshal(map[string]interface{}{"htm": htm, "htu": htu, "iat": iat})
	h := base64.RawURLEncoding.EncodeToString(header)
	p := base64.RawURLEncoding.EncodeToString(payload)
	return h + "." + p + ".fakesig"
}

func TestValidDPoP(t *testing.T) {
	jwk := json.RawMessage(`{"kty":"RSA","n":"abc","e":"AQAB"}`)
	token := makeDPoP("dpop+jwt", "RS256", jwk, "GET", "https://proxy/resource", time.Now().Unix())
	r := httptest.NewRequest("GET", "https://proxy/resource", nil)
	r.Header.Set("DPoP", token)
	proof, err := Validate(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proof.JWKThumbprint == "" {
		t.Fatal("expected thumbprint")
	}
	if proof.Method != "GET" {
		t.Fatalf("expected GET, got %s", proof.Method)
	}
}

func TestNoDPoP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	proof, err := Validate(r)
	if err != nil || proof != nil {
		t.Fatal("expected nil proof and no error when no DPoP header")
	}
}

func TestBadTyp(t *testing.T) {
	jwk := json.RawMessage(`{"kty":"RSA"}`)
	token := makeDPoP("jwt", "RS256", jwk, "GET", "/", time.Now().Unix())
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("DPoP", token)
	_, err := Validate(r)
	if err == nil || !strings.Contains(err.Error(), "typ") {
		t.Fatalf("expected typ error, got %v", err)
	}
}

func TestMethodMismatch(t *testing.T) {
	jwk := json.RawMessage(`{"kty":"RSA"}`)
	token := makeDPoP("dpop+jwt", "RS256", jwk, "POST", "/", time.Now().Unix())
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("DPoP", token)
	_, err := Validate(r)
	if err == nil || !strings.Contains(err.Error(), "method") {
		t.Fatalf("expected method mismatch, got %v", err)
	}
}

func TestExpired(t *testing.T) {
	jwk := json.RawMessage(`{"kty":"RSA"}`)
	token := makeDPoP("dpop+jwt", "RS256", jwk, "GET", "/", time.Now().Add(-10*time.Minute).Unix())
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("DPoP", token)
	_, err := Validate(r)
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired, got %v", err)
	}
}

func TestJWKThumbprint(t *testing.T) {
	jwk := json.RawMessage(`{"kty":"RSA","n":"test","e":"AQAB"}`)
	tp := JWKThumbprint(jwk)
	if tp == "" {
		t.Fatal("expected non-empty thumbprint")
	}
	// Same input = same thumbprint
	tp2 := JWKThumbprint(jwk)
	if tp != tp2 {
		t.Fatal("thumbprint not deterministic")
	}
	// Different input = different thumbprint
	tp3 := JWKThumbprint(json.RawMessage(`{"kty":"EC"}`))
	if tp == tp3 {
		t.Fatal("different keys should have different thumbprints")
	}
	fmt.Println("thumbprint:", tp)
}
