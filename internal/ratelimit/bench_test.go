package ratelimit

import "testing"

func BenchmarkAllow(b *testing.B) {
	l := New(nil, 1000000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Allow("client", nil)
	}
}
