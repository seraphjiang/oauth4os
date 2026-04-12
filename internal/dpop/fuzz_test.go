package dpop

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// FuzzValidate ensures Validate never panics on arbitrary DPoP headers.
func FuzzValidate(f *testing.F) {
	f.Add("")
	f.Add("not.a.jwt")
	f.Add("aaa.bbb.ccc")
	// Valid-ish base64 segments
	h := base64.RawURLEncoding.EncodeToString([]byte(`{"typ":"dpop+jwt","alg":"ES256","jwk":{"kty":"EC"}}`))
	p := base64.RawURLEncoding.EncodeToString([]byte(`{"htm":"GET","htu":"https://example.com","iat":1700000000}`))
	f.Add(h + "." + p + ".fakesig")
	f.Add("x")
	f.Add(".....")
	f.Fuzz(func(t *testing.T, dpop string) {
		r := httptest.NewRequest("GET", "https://example.com/test", nil)
		if dpop != "" {
			r.Header.Set("DPoP", dpop)
		}
		Validate(r) // must not panic
	})
}

// FuzzJWKThumbprint ensures thumbprint never panics.
func FuzzJWKThumbprint(f *testing.F) {
	f.Add([]byte(`{"kty":"EC","crv":"P-256","x":"abc","y":"def"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`"not an object"`))
	f.Add([]byte(`[]`))
	f.Fuzz(func(t *testing.T, jwk []byte) {
		JWKThumbprint(json.RawMessage(jwk)) // must not panic
	})
}
