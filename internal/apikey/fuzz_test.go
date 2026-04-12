package apikey

import (
	"net/http/httptest"
	"testing"
)

// FuzzValidate ensures Validate never panics on arbitrary keys.
func FuzzValidate(f *testing.F) {
	s := NewStore()
	raw, _ := s.Generate("app", []string{"read"})
	f.Add(raw)
	f.Add("")
	f.Add("not-a-key")
	f.Add("sk_0000000000000000000000000000000000000000")
	f.Fuzz(func(t *testing.T, key string) {
		s.Validate(key) // must not panic
	})
}

// FuzzExtractKey ensures header extraction never panics.
func FuzzExtractKey(f *testing.F) {
	f.Add("Bearer sk_abc123")
	f.Add("ApiKey sk_abc123")
	f.Add("")
	f.Add("garbage")
	f.Fuzz(func(t *testing.T, auth string) {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Authorization", auth)
		ExtractKey(r) // must not panic
	})
}
