package pkce

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// FuzzExchange tests PKCE code exchange with random inputs.
// Run: go test -fuzz=FuzzExchange -fuzztime=30s ./internal/pkce/
func FuzzExchange(f *testing.F) {
	f.Add("code=abc&code_verifier=xyz&redirect_uri=http://localhost/cb")
	f.Add("")
	f.Add("code=&code_verifier=&redirect_uri=")
	f.Add(string(make([]byte, 10000)))

	f.Fuzz(func(t *testing.T, form string) {
		h := NewHandler(func(clientID string, scopes []string) (string, string) {
			return "tok", "rtk"
		}, func(clientID, uri string) bool { return true })
		r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		// Must not panic
		h.Exchange(w, r)
	})
}
