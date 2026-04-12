package secrets

import (
	"os"
	"testing"
)

func BenchmarkResolve_Env(b *testing.B) {
	os.Setenv("BENCH_SECRET", "value")
	defer os.Unsetenv("BENCH_SECRET")
	r := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Resolve("env:BENCH_SECRET")
	}
}

func BenchmarkResolve_Plain(b *testing.B) {
	r := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Resolve("plain-value")
	}
}
