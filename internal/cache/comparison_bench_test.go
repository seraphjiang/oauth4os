package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// BenchmarkWithCache measures handler throughput with caching enabled.
func BenchmarkWithCache(b *testing.B) {
	c := New(5*time.Second, 10000)
	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"hits":{"total":{"value":42}}}`))
	})
	// Wrap with cache middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		if e := c.Get(key); e != nil {
			w.WriteHeader(e.StatusCode)
			w.Write(e.Body)
			return
		}
		rec := httptest.NewRecorder()
		backend.ServeHTTP(rec, r)
		c.Set(key, rec.Code, nil, rec.Body.Bytes())
		w.WriteHeader(rec.Code)
		w.Write(rec.Body.Bytes())
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/logs/_search", nil))
	}
}

// BenchmarkWithoutCache measures handler throughput without caching.
func BenchmarkWithoutCache(b *testing.B) {
	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"hits":{"total":{"value":42}}}`))
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		backend.ServeHTTP(w, httptest.NewRequest("GET", "/logs/_search", nil))
	}
}
