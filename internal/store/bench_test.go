package store

import (
	"strconv"
	"testing"
)

func BenchmarkMemorySet(b *testing.B) {
	m := NewMemory()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Set("key-"+strconv.Itoa(i%100), []byte(`"value"`))
	}
}

func BenchmarkMemoryGet(b *testing.B) {
	m := NewMemory()
	m.Set("key", []byte(`"value"`))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Get("key")
	}
}
