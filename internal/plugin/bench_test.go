package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkAuthorize_Empty(b *testing.B) {
	reg := NewRegistry()
	r := httptest.NewRequest("GET", "/", nil)
	claims := map[string]interface{}{"sub": "user"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.Authorize(r, claims)
	}
}

func BenchmarkAuthorize_OnePlugin(b *testing.B) {
	reg := NewRegistry()
	reg.Register(&allowAll{})
	r := httptest.NewRequest("GET", "/", nil)
	claims := map[string]interface{}{"sub": "user"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.Authorize(r, claims)
	}
}

type allowAll struct{}

func (a *allowAll) Name() string { return "allow" }
func (a *allowAll) Authorize(_ *http.Request, _ map[string]interface{}) error { return nil }
