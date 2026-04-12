package plugin

import (
	"net/http/httptest"
	"testing"
)

// FuzzAuthorize ensures plugin registry never panics on arbitrary claims.
func FuzzAuthorize(f *testing.F) {
	f.Add("user", "read write")
	f.Add("", "")
	f.Add("admin", "admin:*")
	f.Fuzz(func(t *testing.T, sub, scope string) {
		reg := NewRegistry()
		r := httptest.NewRequest("GET", "/", nil)
		claims := map[string]interface{}{"sub": sub, "scope": scope}
		reg.Authorize(r, claims) // must not panic
		_ = r
	})
}
