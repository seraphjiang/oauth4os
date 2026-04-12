package circuit

import (
	"testing"
	"time"
)

func BenchmarkAllowClosed(b *testing.B) {
	br := New(100, 1*time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Allow()
	}
}

func BenchmarkAllowOpen(b *testing.B) {
	br := New(1, 10*time.Second)
	br.Record(500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Allow()
	}
}

func BenchmarkRecord(b *testing.B) {
	br := New(1000000, 1*time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Record(200)
	}
}
