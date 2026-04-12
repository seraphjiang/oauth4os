package loadshed

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkAllow(b *testing.B) {
	s := New(1000000)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := s.Middleware(inner)
	r := httptest.NewRequest("GET", "/", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}
