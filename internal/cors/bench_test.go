package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkMiddleware measures CORS middleware overhead.
func BenchmarkMiddleware(b *testing.B) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := Middleware(Config{Origins: []string{"https://app.example.com"}})(inner)
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "https://app.example.com")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}

// BenchmarkPreflight measures OPTIONS preflight overhead.
func BenchmarkPreflight(b *testing.B) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := Middleware(Config{})(inner)
	r := httptest.NewRequest("OPTIONS", "/", nil)
	r.Header.Set("Origin", "https://example.com")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}
