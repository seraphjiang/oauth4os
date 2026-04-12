package circuit

import (
	"testing"
	"time"
)

func BenchmarkAllow_Closed(b *testing.B) {
	br := New(5, 30*time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Allow()
	}
}

func BenchmarkAllow_Open(b *testing.B) {
	br := New(5, 30*time.Second)
	for i := 0; i < 5; i++ {
		br.Record(500)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Allow()
	}
}

func BenchmarkRecord_Success(b *testing.B) {
	br := New(5, 30*time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Record(200)
	}
}

func BenchmarkRecord_Failure(b *testing.B) {
	br := New(1000, 30*time.Second) // high threshold so it doesn't open
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Record(500)
	}
}

func BenchmarkAllow_Concurrent(b *testing.B) {
	br := New(5, 30*time.Second)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			br.Allow()
			br.Record(200)
		}
	})
}
