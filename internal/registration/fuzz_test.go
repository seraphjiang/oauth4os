package registration

import (
	"bytes"
	"net/http/httptest"
	"testing"
)

// FuzzRegister ensures Register never panics on arbitrary JSON input.
func FuzzRegister(f *testing.F) {
	f.Add([]byte(`{"client_name":"test"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"client_name":"","scope":"admin read:logs-*"}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[1,2,3]`))
	f.Add([]byte(``))
	f.Add([]byte(`{"client_name":123}`))
	f.Add([]byte(`{"client_name":"x","redirect_uris":null,"grant_types":["authorization_code"]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		h := NewHandler(func(id, secret string, scopes, redirectURIs []string) {}, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/oauth/register", bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		h.Register(w, r)
		// Must not panic; any status code is fine
	})
}
