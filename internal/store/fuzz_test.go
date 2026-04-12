package store

import "testing"

// FuzzMemorySetGet ensures Memory store never panics on arbitrary keys/values.
func FuzzMemorySetGet(f *testing.F) {
	f.Add("key", "value")
	f.Add("", "")
	f.Add("k\x00ey", "v\x00al")
	f.Add("a/b/c", `{"json":true}`)
	f.Fuzz(func(t *testing.T, key, value string) {
		m := NewMemory()
		m.Set(key, []byte(value))
		m.Get(key)
		m.Delete(key)
		m.List()
		// must not panic
	})
}
