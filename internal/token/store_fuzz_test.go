package token

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzClientStoreLoad ensures loading arbitrary JSON never panics.
func FuzzClientStoreLoad(f *testing.F) {
	f.Add([]byte(`[{"ID":"app","Secret":"s","Scopes":["read"]}]`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(`[{"ID":""}]`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "clients.json")
		os.WriteFile(path, data, 0644)
		mgr := NewManager()
		NewClientStore(path, mgr) // must not panic
	})
}
