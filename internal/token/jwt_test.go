package token

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

type testKeyProvider struct {
	key *rsa.PrivateKey
}

func (p *testKeyProvider) CurrentKey() (string, *rsa.PrivateKey) {
	return "test-kid-1", p.key
}

func TestJWTAccessToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	m.EnableJWT("https://auth.example.com", &testKeyProvider{key: key})
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)

	tok, refresh := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})

	// Token ID should be a JWT (3 dot-separated parts)
	parts := strings.Split(tok.ID, ".")
	if len(parts) != 3 {
		t.Fatalf("expected JWT with 3 parts, got %d", len(parts))
	}

	// Decode header
	headerBytes, _ := base64.RawURLEncoding.DecodeString(parts[0])
	var header map[string]string
	json.Unmarshal(headerBytes, &header)
	if header["alg"] != "RS256" {
		t.Fatalf("expected RS256, got %s", header["alg"])
	}
	if header["typ"] != "at+jwt" {
		t.Fatalf("expected at+jwt, got %s", header["typ"])
	}
	if header["kid"] != "test-kid-1" {
		t.Fatalf("expected test-kid-1, got %s", header["kid"])
	}

	// Decode payload
	payloadBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]interface{}
	json.Unmarshal(payloadBytes, &claims)
	if claims["iss"] != "https://auth.example.com" {
		t.Fatalf("expected issuer, got %v", claims["iss"])
	}
	if claims["sub"] != "svc-1" {
		t.Fatalf("expected sub=svc-1, got %v", claims["sub"])
	}
	if claims["scope"] != "read:logs-*" {
		t.Fatalf("expected scope, got %v", claims["scope"])
	}

	// Refresh token should still be opaque
	if strings.Contains(refresh, ".") {
		t.Fatal("refresh token should remain opaque")
	}

	// Token should be valid in the store
	if !m.IsValid(tok.ID) {
		t.Fatal("JWT token should be valid in store")
	}
}

func TestOpaqueTokenWhenJWTDisabled(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)

	tok, _ := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})

	if strings.Contains(tok.ID, ".") {
		t.Fatal("opaque token should not contain dots")
	}
	if !strings.HasPrefix(tok.ID, "tok_") {
		t.Fatalf("expected tok_ prefix, got %s", tok.ID[:10])
	}
}

func TestJWTSignatureValid(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	m.EnableJWT("https://auth.example.com", &testKeyProvider{key: key})
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)

	tok, _ := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})
	parts := strings.Split(tok.ID, ".")
	if len(parts) != 3 {
		t.Fatal("not a JWT")
	}

	// Verify signature with public key
	sigInput := []byte(parts[0] + "." + parts[1])
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(sigInput)
	err = rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, h[:], sig)
	if err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
}
