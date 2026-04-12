package jwt

import (
	"sync"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

func TestConcurrentValidate(t *testing.T) {
	v := NewValidator([]config.Provider{
		{Name: "test", Issuer: "https://auth.example.com", JWKSURI: "https://auth.example.com/.well-known/jwks.json"},
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Invalid token — should error, not panic
			_, err := v.Validate("eyJhbGciOiJSUzI1NiJ9.eyJpc3MiOiJodHRwczovL2F1dGguZXhhbXBsZS5jb20ifQ.invalid")
			if err == nil {
				t.Error("invalid signature should fail")
			}
		}()
	}
	wg.Wait()
}

func TestEmptyTokenRejected(t *testing.T) {
	v := NewValidator(nil)
	_, err := v.Validate("")
	if err == nil {
		t.Fatal("empty token should fail")
	}
}

func TestMalformedToken(t *testing.T) {
	v := NewValidator(nil)
	_, err := v.Validate("not-a-jwt")
	if err == nil {
		t.Fatal("malformed token should fail")
	}
}

func TestTwoPartToken(t *testing.T) {
	v := NewValidator(nil)
	_, err := v.Validate("header.payload")
	if err == nil {
		t.Fatal("2-part token should fail")
	}
}

func TestUnknownIssuerRejected(t *testing.T) {
	v := NewValidator([]config.Provider{
		{Name: "test", Issuer: "https://known.example.com"},
	})
	// Token with unknown issuer
	_, err := v.Validate("eyJhbGciOiJSUzI1NiJ9.eyJpc3MiOiJodHRwczovL3Vua25vd24uZXhhbXBsZS5jb20ifQ.sig")
	if err == nil {
		t.Fatal("unknown issuer should fail")
	}
}
