package backup

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

// FuzzImport ensures Import never panics on arbitrary JSON input.
func FuzzImport(f *testing.F) {
	f.Add([]byte(`{"version":"1","providers":[]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(``))
	f.Add([]byte(`[1,2,3]`))
	f.Add([]byte(`{"clients":[{"client_id":"x","scopes":["admin"]}]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		h := NewHandler(
			func() *config.Config { return &config.Config{} },
			func() []ClientEntry { return nil },
			func(c *config.Config) {},
		)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/admin/backup/import", bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		h.Import(w, r)
	})
}
