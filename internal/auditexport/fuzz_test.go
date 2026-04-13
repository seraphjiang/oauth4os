package auditexport

import (
	"encoding/json"
	"testing"
)

// FuzzAdd ensures Add never panics on arbitrary JSON.
func FuzzAdd(f *testing.F) {
	f.Add(`{"action":"login"}`)
	f.Add(`{}`)
	f.Add(`null`)
	f.Add(``)
	f.Add(`not json at all`)
	f.Fuzz(func(t *testing.T, entry string) {
		u := &memUploader{data: map[string][]byte{}}
		e := New(u, "audit/", 0)
		defer e.Stop()
		e.Add(json.RawMessage(entry)) // must not panic
	})
}
