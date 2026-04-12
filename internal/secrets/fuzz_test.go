package secrets

import "testing"

// FuzzResolve ensures Resolve never panics on arbitrary ref strings.
func FuzzResolve(f *testing.F) {
	f.Add("env:HOME")
	f.Add("file:/etc/hostname")
	f.Add("plain-value")
	f.Add("")
	f.Add("env:")
	f.Add("file:")
	f.Add("unknown-scheme:key")
	f.Add("env:NONEXISTENT_VAR_XYZ")
	f.Fuzz(func(t *testing.T, ref string) {
		r := New()
		r.Resolve(ref) // must not panic
	})
}
