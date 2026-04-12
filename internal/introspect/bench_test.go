package introspect

import (
	"testing"
	"time"
)

type stubLookup struct {
	resp *Response
}

func (s *stubLookup) Introspect(token string) *Response { return s.resp }

func BenchmarkDirectLookup(b *testing.B) {
	inner := &stubLookup{resp: &Response{Active: true, ClientID: "svc-1"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inner.Introspect("tok-1")
	}
}

func BenchmarkCachedLookup(b *testing.B) {
	inner := &stubLookup{resp: &Response{Active: true, ClientID: "svc-1"}}
	cached := NewCachedLookup(inner, 30*time.Second)
	// Warm cache
	cached.Introspect("tok-1")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cached.Introspect("tok-1")
	}
}

func BenchmarkCachedLookupMiss(b *testing.B) {
	inner := &stubLookup{resp: &Response{Active: true, ClientID: "svc-1"}}
	cached := NewCachedLookup(inner, 1*time.Nanosecond) // always expired
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cached.Introspect("tok-1")
	}
}
