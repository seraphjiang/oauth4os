package userinfo

import (
	"net/http/httptest"
	"testing"
)

// FuzzServeHTTP ensures userinfo handler never panics on arbitrary auth headers.
func FuzzServeHTTP(f *testing.F) {
	f.Add("Bearer valid-tok")
	f.Add("Bearer invalid")
	f.Add("")
	f.Add("Basic dXNlcjpwYXNz")
	f.Add("Bearer " + string(make([]byte, 10000)))
	f.Fuzz(func(t *testing.T, auth string) {
		h := New(stubLookup)
		r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
		if auth != "" {
			r.Header.Set("Authorization", auth)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r) // must not panic
	})
}
