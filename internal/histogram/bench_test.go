package histogram

import (
	"testing"
	"time"
)

func BenchmarkObserve_WithPath(b *testing.B) {
	h := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Observe(50*time.Millisecond, "/api/query")
	}
}

func BenchmarkObserve_NoPath(b *testing.B) {
	h := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Observe(50*time.Millisecond, "")
	}
}

func BenchmarkWritePrometheus(b *testing.B) {
	h := New()
	for i := 0; i < 1000; i++ {
		h.Observe(time.Duration(i)*time.Millisecond, "/api/query")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.WritePrometheus(discard{}, "http_request_duration")
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
