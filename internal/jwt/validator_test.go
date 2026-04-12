package jwt

import (
	"testing"

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
