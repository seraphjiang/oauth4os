package jwt

import (
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

// Mutation tests for jwt package — pure function logic.

// M1: extractScopes with space-delimited string.
func TestMutation_ExtractScopesString(t *testing.T) {
	claims := jwtgo.MapClaims{"scope": "read write admin"}
	s := extractScopes(claims)
	if len(s) != 3 || s[0] != "read" || s[1] != "write" || s[2] != "admin" {
		t.Fatalf("expected [read write admin], got %v", s)
	}
}

// M2: extractScopes with array.
func TestMutation_ExtractScopesArray(t *testing.T) {
	claims := jwtgo.MapClaims{"scope": []interface{}{"read", "write"}}
	s := extractScopes(claims)
	if len(s) != 2 || s[0] != "read" || s[1] != "write" {
		t.Fatalf("expected [read write], got %v", s)
	}
}

// M3: extractScopes with missing scope returns nil.
func TestMutation_ExtractScopesNil(t *testing.T) {
	claims := jwtgo.MapClaims{}
	s := extractScopes(claims)
	if s != nil {
		t.Fatalf("expected nil, got %v", s)
	}
}

// M4: extractScopes with non-string array elements skips them.
func TestMutation_ExtractScopesArrayMixed(t *testing.T) {
	claims := jwtgo.MapClaims{"scope": []interface{}{"read", 42, "write"}}
	s := extractScopes(claims)
	if len(s) != 2 || s[0] != "read" || s[1] != "write" {
		t.Fatalf("expected [read write], got %v", s)
	}
}

// M5: audienceMatch — match found.
func TestMutation_AudienceMatchFound(t *testing.T) {
	if !audienceMatch([]string{"a", "b"}, []string{"b", "c"}) {
		t.Fatal("expected match on 'b'")
	}
}

// M6: audienceMatch — no match.
func TestMutation_AudienceMatchNotFound(t *testing.T) {
	if audienceMatch([]string{"a"}, []string{"b", "c"}) {
		t.Fatal("expected no match")
	}
}

// M7: audienceMatch — empty token audience.
func TestMutation_AudienceMatchEmptyToken(t *testing.T) {
	if audienceMatch(nil, []string{"b"}) {
		t.Fatal("nil audience should not match")
	}
}

// M8: audienceMatch — empty expected.
func TestMutation_AudienceMatchEmptyExpected(t *testing.T) {
	if audienceMatch([]string{"a"}, nil) {
		t.Fatal("nil expected should not match")
	}
}

// M9: findProvider — match.
func TestMutation_FindProviderMatch(t *testing.T) {
	v := NewValidator([]config.Provider{
		{Issuer: "https://a.com"},
		{Issuer: "https://b.com"},
	})
	p := v.findProvider("https://b.com")
	if p == nil || p.Issuer != "https://b.com" {
		t.Fatal("expected provider b.com")
	}
}

// M10: findProvider — no match.
func TestMutation_FindProviderNoMatch(t *testing.T) {
	v := NewValidator([]config.Provider{{Issuer: "https://a.com"}})
	if v.findProvider("https://unknown.com") != nil {
		t.Fatal("expected nil for unknown issuer")
	}
}

// M11: findKey — kid match.
func TestMutation_FindKeyByKid(t *testing.T) {
	keys := []jwksKey{
		{Kid: "k1", Kty: "RSA", N: "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw", E: "AQAB"},
		{Kid: "k2", Kty: "RSA", N: "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw", E: "AQAB"},
	}
	key, err := findKey(keys, "k2")
	if err != nil || key == nil {
		t.Fatalf("expected key k2, got err: %v", err)
	}
}

// M12: findKey — kid not found.
func TestMutation_FindKeyMissing(t *testing.T) {
	keys := []jwksKey{{Kid: "k1", Kty: "RSA", N: "AQAB", E: "AQAB"}}
	_, err := findKey(keys, "k99")
	if err == nil {
		t.Fatal("expected error for missing kid")
	}
}

// M13: findKey — fallback when kid is empty (use first RSA key).
func TestMutation_FindKeyFallbackNoKid(t *testing.T) {
	keys := []jwksKey{
		{Kid: "k1", Kty: "RSA", N: "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw", E: "AQAB"},
	}
	key, err := findKey(keys, "")
	if err != nil || key == nil {
		t.Fatalf("expected fallback key, got err: %v", err)
	}
}

// M14: findKey — non-RSA keys skipped.
func TestMutation_FindKeySkipsNonRSA(t *testing.T) {
	keys := []jwksKey{{Kid: "k1", Kty: "EC"}}
	_, err := findKey(keys, "k1")
	if err == nil {
		t.Fatal("EC key should not match RSA lookup")
	}
}

// M15: Validate rejects empty token.
func TestMutation_ValidateEmptyToken(t *testing.T) {
	v := NewValidator(nil)
	_, err := v.Validate("")
	if err == nil {
		t.Fatal("empty token must be rejected")
	}
}

// M16: parseRSAKey with invalid base64.
func TestMutation_ParseRSAKeyInvalidN(t *testing.T) {
	_, err := parseRSAKey(jwksKey{N: "!!!invalid!!!", E: "AQAB"})
	if err == nil {
		t.Fatal("invalid N should fail")
	}
}

// M17: parseRSAKey with invalid E.
func TestMutation_ParseRSAKeyInvalidE(t *testing.T) {
	_, err := parseRSAKey(jwksKey{N: "AQAB", E: "!!!invalid!!!"})
	if err == nil {
		t.Fatal("invalid E should fail")
	}
}
