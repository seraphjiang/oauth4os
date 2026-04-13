package userinfo

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkServeHTTP_Valid(b *testing.B) {
	h := New(stubLookup)
	r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
	r.Header.Set("Authorization", "Bearer valid-tok")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}
}

func BenchmarkServeHTTP_Invalid(b *testing.B) {
	h := New(stubLookup)
	r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
	r.Header.Set("Authorization", "Bearer bad")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}
}
