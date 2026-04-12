package introspect

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// FuzzIntrospect ensures the introspect handler never panics.
func FuzzIntrospect(f *testing.F) {
	f.Add("token=valid-token")
	f.Add("")
	f.Add("token=")
	f.Add("token=" + strings.Repeat("x", 10000))
	f.Fuzz(func(t *testing.T, body string) {
		h := NewHandler(&StaticLookup{})
		r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r) // must not panic
	})
}
