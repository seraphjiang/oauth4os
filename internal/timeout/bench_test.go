package timeout

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func BenchmarkMiddleware_Fast(b *testing.B) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	handler := Middleware(5*time.Second)(inner)
	r := httptest.NewRequest("GET", "/", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}
