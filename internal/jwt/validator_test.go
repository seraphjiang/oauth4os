package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/seraphjiang/oauth4os/internal/config"
)

func TestEmptyToken(t *testing.T) {
	v := NewValidator(nil)
	_, err := v.Validate("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestMalformedTokens(t *testing.T) {
	v := NewValidator(nil)
	for _, tok := range []string{
		"not-a-jwt",
		"a.b",
		"a.b.c.d",
		".....",
		" ",
	} {
		_, err := v.Validate(tok)
		if err == nil {
			t.Fatalf("expected error for malformed token: %q", tok)
		}
	}
}

func TestUnknownIssuer(t *testing.T) {
	v := NewValidator([]config.Provider{
		{Name: "known", Issuer: "https://known.example.com"},
	})
	// base64url({"alg":"RS256"}).base64url({"iss":"https://evil.com","exp":9999999999}).fakesig
	_, err := v.Validate("eyJhbGciOiJSUzI1NiJ9.eyJpc3MiOiJodHRwczovL2V2aWwuY29tIiwiZXhwIjo5OTk5OTk5OTk5fQ.fakesig")
	if err == nil {
		t.Fatal("expected error for unknown issuer")
	}
}

func TestExtractScopesString(t *testing.T) {
	claims := jwtgo.MapClaims{"scope": "read:logs-* write:dashboards admin"}
	scopes := extractScopes(claims)
	if len(scopes) != 3 {
		t.Fatalf("expected 3 scopes, got %d: %v", len(scopes), scopes)
	}
	if scopes[0] != "read:logs-*" || scopes[1] != "write:dashboards" || scopes[2] != "admin" {
		t.Fatalf("unexpected scopes: %v", scopes)
	}
}

func TestExtractScopesArray(t *testing.T) {
	claims := jwtgo.MapClaims{"scope": []interface{}{"read:logs-*", "admin"}}
	scopes := extractScopes(claims)
	if len(scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %d: %v", len(scopes), scopes)
	}
}

func TestExtractScopesEmpty(t *testing.T) {
	claims := jwtgo.MapClaims{}
	scopes := extractScopes(claims)
	if len(scopes) != 0 {
		t.Fatalf("expected 0 scopes, got %d", len(scopes))
	}
}

func TestExtractScopesNonStringType(t *testing.T) {
	claims := jwtgo.MapClaims{"scope": 42}
	scopes := extractScopes(claims)
	if len(scopes) != 0 {
		t.Fatalf("expected 0 scopes for non-string, got %d", len(scopes))
	}
}

func TestFindKeyNoMatch(t *testing.T) {
	keys := []jwksKey{
		{Kid: "key-1", Kty: "RSA", N: "abc", E: "AQAB"},
	}
	_, err := findKey(keys, "key-999")
	if err == nil {
		t.Fatal("expected error for missing kid")
	}
}

func TestFindKeyFallbackNoKid(t *testing.T) {
	keys := []jwksKey{
		{Kid: "key-1", Kty: "EC"},
		{Kid: "key-2", Kty: "RSA",
			N: "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
			E: "AQAB"},
	}
	key, err := findKey(keys, "")
	if err != nil {
		t.Fatalf("expected fallback to RSA key, got: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil key")
	}
}

func TestHTTPClientHasTimeout(t *testing.T) {
	v := NewValidator(nil)
	if v.client.Timeout == 0 {
		t.Fatal("HTTP client should have a timeout")
	}
}

func TestFindProviderMiss(t *testing.T) {
	v := NewValidator([]config.Provider{
		{Name: "a", Issuer: "https://a.example.com"},
	})
	if v.findProvider("https://b.example.com") != nil {
		t.Fatal("expected nil for unknown issuer")
	}
}

func TestFindProviderHit(t *testing.T) {
	v := NewValidator([]config.Provider{
		{Name: "a", Issuer: "https://a.example.com"},
	})
	p := v.findProvider("https://a.example.com")
	if p == nil || p.Name != "a" {
		t.Fatal("expected provider 'a'")
	}
}

func TestAudienceMatch(t *testing.T) {
	if !audienceMatch([]string{"api.example.com"}, []string{"api.example.com", "other"}) {
		t.Fatal("expected match")
	}
	if audienceMatch([]string{"evil.com"}, []string{"api.example.com"}) {
		t.Fatal("expected no match")
	}
	if audienceMatch(nil, []string{"api.example.com"}) {
		t.Fatal("expected no match for nil aud")
	}
	if audienceMatch([]string{"a", "b"}, []string{"c"}) {
		t.Fatal("expected no match")
	}
}

func TestGetJWKS(t *testing.T) {
	jwks := `{"keys":[{"kty":"RSA","kid":"test-key","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw","e":"AQAB","use":"sig","alg":"RS256"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jwks))
	}))
	defer srv.Close()

	provider := &config.Provider{Name: "test", Issuer: "https://test.example.com", JWKSURI: srv.URL}
	v := NewValidator([]config.Provider{*provider})
	keys, err := v.getJWKS(provider, true)
	if err != nil {
		t.Fatalf("getJWKS failed: %v", err)
	}
	if len(keys) == 0 {
		t.Fatal("expected at least 1 key")
	}
}

func TestResolveJWKSURI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"issuer":%q,"jwks_uri":"https://example.com/.well-known/jwks.json"}`, "http://"+r.Host)
	}))
	defer srv.Close()

	provider := &config.Provider{Name: "test", Issuer: srv.URL, JWKSURI: "auto"}
	v := NewValidator([]config.Provider{*provider})
	uri, err := v.resolveJWKSURI(provider)
	if err != nil {
		t.Fatalf("resolveJWKSURI failed: %v", err)
	}
	if uri != "https://example.com/.well-known/jwks.json" {
		t.Errorf("expected jwks_uri, got %q", uri)
	}
}

func TestResolveJWKSURI_Failure(t *testing.T) {
	provider := &config.Provider{Name: "bad", Issuer: "http://localhost:1", JWKSURI: "auto"}
	v := NewValidator([]config.Provider{*provider})
	_, err := v.resolveJWKSURI(provider)
	if err == nil {
		t.Error("expected error for unreachable issuer")
	}
}

func TestValidateFullFlow(t *testing.T) {
	// Generate RSA key pair
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	pubKey := &privKey.PublicKey

	// Serve JWKS
	n := base64.RawURLEncoding.EncodeToString(pubKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pubKey.E)).Bytes())
	jwksJSON := fmt.Sprintf(`{"keys":[{"kty":"RSA","kid":"k1","n":"%s","e":"%s","use":"sig","alg":"RS256"}]}`, n, e)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jwksJSON))
	}))
	defer srv.Close()

	// Create a signed JWT
	now := time.Now()
	claims := jwtgo.MapClaims{
		"iss":       "https://test-idp.example.com",
		"sub":       "user-123",
		"client_id": "my-client",
		"scope":     "read:logs-* write:logs-app",
		"exp":       now.Add(time.Hour).Unix(),
		"iat":       now.Unix(),
	}
	tok := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, claims)
	tok.Header["kid"] = "k1"
	tokenStr, err := tok.SignedString(privKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	// Validate
	v := NewValidator([]config.Provider{{
		Name:    "test",
		Issuer:  "https://test-idp.example.com",
		JWKSURI: srv.URL,
	}})
	result, err := v.Validate(tokenStr)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if result.ClientID != "my-client" {
		t.Errorf("ClientID = %q", result.ClientID)
	}
	if result.Subject != "user-123" {
		t.Errorf("Subject = %q", result.Subject)
	}
	if len(result.Scopes) != 2 {
		t.Errorf("Scopes = %v", result.Scopes)
	}
}

func TestResolveKeyRotation(t *testing.T) {
	// First serve old key, then new key on refresh
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	pubKey := &privKey.PublicKey
	n := base64.RawURLEncoding.EncodeToString(pubKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pubKey.E)).Bytes())

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		// Always return the key with kid "new-key"
		fmt.Fprintf(w, `{"keys":[{"kty":"RSA","kid":"new-key","n":"%s","e":"%s","use":"sig","alg":"RS256"}]}`, n, e)
	}))
	defer srv.Close()

	provider := &config.Provider{Name: "test", Issuer: "https://test.example.com", JWKSURI: srv.URL}
	v := NewValidator([]config.Provider{*provider})

	// Request with kid "new-key" — should find it
	key, err := v.resolveKey(provider, "new-key")
	if err != nil {
		t.Fatalf("resolveKey: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil key")
	}

	// Request with unknown kid — should trigger refresh retry
	_, err = v.resolveKey(provider, "unknown-kid")
	if err == nil {
		t.Error("expected error for unknown kid")
	}
	if calls < 2 {
		t.Error("expected JWKS refresh retry")
	}
}
