package cache

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkGet_Hit(b *testing.B) {
	c := New(5*time.Second, 1000)
	c.Set("key", 200, nil, []byte(`{"ok":true}`))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("key")
	}
}

func BenchmarkGet_Miss(b *testing.B) {
	c := New(5*time.Second, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("missing")
	}
}

func BenchmarkSet(b *testing.B) {
	c := New(5*time.Second, 10000)
	body := []byte(`{"hits":{"total":100}}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set(fmt.Sprintf("key-%d", i), 200, nil, body)
	}
}

func BenchmarkGet_Concurrent(b *testing.B) {
	c := New(5*time.Second, 1000)
	for i := 0; i < 100; i++ {
		c.Set(fmt.Sprintf("k%d", i), 200, nil, []byte("data"))
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(fmt.Sprintf("k%d", i%100))
			i++
		}
	})
}

func BenchmarkSetGet_Mixed(b *testing.B) {
	c := New(5*time.Second, 1000)
	body := []byte(`{"data":"value"}`)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("k%d", i%50)
			if i%5 == 0 {
				c.Set(key, 200, nil, body)
			} else {
				c.Get(key)
			}
			i++
		}
	})
}
