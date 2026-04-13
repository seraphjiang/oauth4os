package ratelimit

import "testing"

func BenchmarkAllow(b *testing.B) {
	l := New(nil, 6000)
	scopes := []string{"read"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Allow("client-1", scopes)
	}
}

func BenchmarkAllow_NewClient(b *testing.B) {
	l := New(nil, 6000)
	scopes := []string{"read"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Allow("client-"+string(rune(i%1000)), scopes)
	}
}
