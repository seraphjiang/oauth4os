package circuit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// BenchmarkWithCircuitBreaker measures handler throughput with circuit breaker.
func BenchmarkWithCircuitBreaker(b *testing.B) {
	br := New(100, 5*time.Second)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	handler := br.Middleware(inner)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	}
}

// BenchmarkWithoutCircuitBreaker measures raw handler throughput.
func BenchmarkWithoutCircuitBreaker(b *testing.B) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		inner.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	}
}
