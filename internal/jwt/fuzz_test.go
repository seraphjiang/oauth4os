package jwt

import (
	"strings"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

// FuzzValidate ensures the JWT validator never panics on arbitrary tokens.
func FuzzValidate(f *testing.F) {
	f.Add("eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0.fakesig")
	f.Add("")
	f.Add("not-a-jwt")
	f.Add("a.b.c")
	f.Add(strings.Repeat("A", 10000))
	f.Add("eyJhbGciOiJub25lIn0.eyJzdWIiOiJoYWNrZXIifQ.")
	f.Fuzz(func(t *testing.T, token string) {
		v := NewValidator([]config.Provider{{
			Name:   "test",
			Issuer: "https://test.example.com",
		}})
		v.Validate(token) // must not panic
	})
}

// FuzzExtractScopes ensures scope extraction never panics.
func FuzzExtractScopes(f *testing.F) {
	f.Add("read write admin")
	f.Add("")
	f.Add("a b c d e f g")
	f.Fuzz(func(t *testing.T, scopeStr string) {
		// Build a MapClaims with the scope string
		claims := map[string]interface{}{"scope": scopeStr}
		extractScopes(claims) // must not panic
	})
}

// FuzzAudienceMatch ensures audience matching never panics.
func FuzzAudienceMatch(f *testing.F) {
	f.Add("aud1", "aud1")
	f.Add("", "")
	f.Add("a,b,c", "b")
	f.Fuzz(func(t *testing.T, tokenAud, expected string) {
		ta := strings.Split(tokenAud, ",")
		ea := strings.Split(expected, ",")
		audienceMatch(ta, ea) // must not panic
	})
}
