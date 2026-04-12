package token

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// FuzzIssueToken tests token issuance with random form data.
// Run: go test -fuzz=FuzzIssueToken -fuzztime=30s ./internal/token/
func FuzzIssueToken(f *testing.F) {
	f.Add("grant_type=client_credentials&client_id=app&client_secret=secret&scope=read:logs")
	f.Add("grant_type=refresh_token&client_id=app&client_secret=secret&refresh_token=bogus")
	f.Add("")
	f.Add("grant_type=password")
	f.Add(string(make([]byte, 10000)))
	f.Add("grant_type=client_credentials&client_id=" + strings.Repeat("A", 5000))

	f.Fuzz(func(t *testing.T, form string) {
		m := NewManager()
		m.RegisterClient("app", "secret", []string{"read:logs"}, nil)
		r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		// Must not panic
		m.IssueToken(w, r)
	})
}
