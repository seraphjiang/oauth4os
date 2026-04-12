package idempotency

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func BenchmarkMiddleware_UniqueKeys(b *testing.B) {
	s := New(5 * time.Second)
	defer s.Stop()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := s.Middleware(inner)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Set("Idempotency-Key", "k-"+strconv.Itoa(i))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkMiddleware_CachedKey(b *testing.B) {
	s := New(5 * time.Second)
	defer s.Stop()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	handler := s.Middleware(inner)
	// Prime the cache
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("Idempotency-Key", "cached")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Set("Idempotency-Key", "cached")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}
