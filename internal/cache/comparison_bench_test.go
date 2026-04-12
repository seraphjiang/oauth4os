package cache

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkWithWithoutCache compares direct access vs cached access patterns.
func BenchmarkWithoutCache_DirectLookup(b *testing.B) {
	// Simulate: no cache, every request hits "upstream" (map lookup as proxy)
	data := make(map[string][]byte)
	for i := 0; i < 100; i++ {
		data[fmt.Sprintf("client:/%d/_search", i)] = []byte(`{"hits":{"total":100}}`)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("client:/%d/_search", i%100)
		_ = data[key]
	}
}

func BenchmarkWithCache_CachedLookup(b *testing.B) {
	c := New(5*time.Second, 1000)
	for i := 0; i < 100; i++ {
		c.Set(fmt.Sprintf("client:/%d/_search", i), 200, nil, []byte(`{"hits":{"total":100}}`))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("client:/%d/_search", i%100)
		c.Get(key)
	}
}

func BenchmarkCache_ThroughputComparison(b *testing.B) {
	c := New(5*time.Second, 1000)
	body := []byte(`{"hits":{"total":{"value":42},"hits":[{"_source":{"level":"ERROR"}}]}}`)

	b.Run("no_cache", func(b *testing.B) {
		store := make(map[string][]byte)
		for i := 0; i < 50; i++ {
			store[fmt.Sprintf("k%d", i)] = body
		}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				_ = store[fmt.Sprintf("k%d", i%50)]
				i++
			}
		})
	})

	b.Run("with_cache", func(b *testing.B) {
		for i := 0; i < 50; i++ {
			c.Set(fmt.Sprintf("k%d", i), 200, map[string]string{"Content-Type": "application/json"}, body)
		}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				c.Get(fmt.Sprintf("k%d", i%50))
				i++
			}
		})
	})
}
