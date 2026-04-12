package cache

import (
	"testing"
	"time"
)

func BenchmarkCacheHit(b *testing.B) {
	c := New(5*time.Second, 10000)
	c.Set("/test", 200, nil, []byte("cached response body"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("/test")
	}
}

func BenchmarkCacheMiss(b *testing.B) {
	c := New(5*time.Second, 10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("/nonexistent")
	}
}

func BenchmarkCacheSet(b *testing.B) {
	c := New(5*time.Second, 100000)
	body := []byte("response body data for benchmarking")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set("/bench", 200, nil, body)
	}
}
