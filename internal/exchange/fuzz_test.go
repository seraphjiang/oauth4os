package exchange

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// FuzzExchange ensures the exchange handler never panics on arbitrary form input.
func FuzzExchange(f *testing.F) {
	f.Add("urn:ietf:params:oauth:grant-type:token-exchange", "valid-token", "")
	f.Add("", "", "")
	f.Add("authorization_code", "x", "scope1 scope2")
	f.Add("urn:ietf:params:oauth:grant-type:token-exchange", strings.Repeat("A", 10000), "admin")
	f.Fuzz(func(t *testing.T, grantType, subjectToken, scope string) {
		v := &StaticSubjectValidator{Claims: &SubjectClaims{Subject: "user", Issuer: "https://idp"}}
		i := &StaticTokenIssuer{TokenID: "tok", ExpiresIn: 3600}
		h := NewHandler(v, i, "aud")
		body := "grant_type=" + grantType + "&subject_token=" + subjectToken
		if scope != "" {
			body += "&scope=" + scope
		}
		r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r) // must not panic
	})
}
