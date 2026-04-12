package circuit

import (
	"testing"
	"time"
)

// BenchmarkWithWithoutBreaker compares overhead of circuit breaker check.
func BenchmarkOverhead_NoBreaker(b *testing.B) {
	// Simulate: direct pass-through (no check)
	allowed := true
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if allowed {
			_ = i // simulate work
		}
	}
}

func BenchmarkOverhead_WithBreaker(b *testing.B) {
	br := New(5, 30*time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if br.Allow() {
			br.Record(200)
		}
	}
}

func BenchmarkOverhead_WithBreaker_Concurrent(b *testing.B) {
	br := New(5, 30*time.Second)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if br.Allow() {
				br.Record(200)
			}
		}
	})
}
