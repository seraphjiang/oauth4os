package accesslog

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkMiddleware(b *testing.B) {
	l := New(io.Discard)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	handler := l.Middleware(inner, nil)
	r := httptest.NewRequest("GET", "/logs/_search", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}
