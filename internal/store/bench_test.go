package store

import (
	"path/filepath"
	"strconv"
	"testing"
)

func BenchmarkFileSet(b *testing.B) {
	dir := b.TempDir()
	f, _ := NewFile(filepath.Join(dir, "bench.json"))
	defer f.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Set("key-"+strconv.Itoa(i%100), []byte(`"value"`))
	}
}

func BenchmarkFileGet(b *testing.B) {
	dir := b.TempDir()
	f, _ := NewFile(filepath.Join(dir, "bench.json"))
	defer f.Close()
	f.Set("key", []byte(`"value"`))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Get("key")
	}
}
