package token

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"
)

// testKeyProvider returns a fixed RSA key for testing.
type testKeyProvider struct {
	key *rsa.PrivateKey
}

func (p *testKeyProvider) CurrentKey() (string, *rsa.PrivateKey) {
	return "test-kid-1", p.key
}

func genTestKey() *rsa.PrivateKey {
	block, _ := pem.Decode([]byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA2a2rwplBQLHgcad0jBJwMGC1GGiGMGOSulm0gSMoH7KQMBB
DqoMiMJoKSAFakb0NKZG0axOwSVg0lFSgmYAE5UKMwlMSvBcrUGmYN1qBmmXSxH
qNPv0yCKwFSVMUgT0hU7VEZdQEhJR7bXhIDGCEO1kF2wSh0GUhYCgqJMHG8RTFH
cVQdp0KXMB6RCBMZ0bPOINOLQJkGzmHHBJ8JfCEGRYMzLOTDmmna0MZWBPMNQFN
fy6MjSYBJIliTvBcBl0E0QKFKGUZ/gAEJPMTAS0MFzMfFhXJZBP1kHiJJoZXhNY5
q0bFTvBKhJcnHGNMBuSNQ1PF+QCmSdPsOzMNNQIDAQABAoIBAC5RgZ+hBx7xHNaM
pPgwGMnCd2vHoqFMBGIkf0Ov3K3G3gR7FERQ0nSGMjJbGMN0QPKBM5caN1JJCCE
kN3cLlpMSAdYBNS4p0O3OFv0IAZuHLpgJLYDiG7aYJXYyzsCAwEAAQ==
-----END RSA PRIVATE KEY-----`))
	if block == nil {
		// Generate a fresh key if PEM fails
		key, _ := rsa.GenerateKey(strings.NewReader("deterministic-seed-for-test-only!!"), 2048)
		return key
	}
	key, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
	return key
}

func TestJWTAccessToken(t *testing.T) {
	key, _ := rsa.GenerateKey(strings.NewReader("deterministic-seed-for-test-only!!deterministic-seed-for-test-only!!"), 2048)
	if key == nil {
		t.Skip("cannot generate RSA key")
	}
	m := NewManager()
	m.EnableJWT("https://auth.example.com", &testKeyProvider{key: key})
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)

	tok, refresh := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})

	// Token ID should be a JWT (3 dot-separated parts)
	parts := strings.Split(tok.ID, ".")
	if len(parts) != 3 {
		t.Fatalf("expected JWT with 3 parts, got %d: %s", len(parts), tok.ID[:min(50, len(tok.ID))])
	}

	// Decode header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("invalid base64 header: %v", err)
	}
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
