package tokenbind

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkFingerprint(b *testing.B) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("User-Agent", "Mozilla/5.0")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Fingerprint(r)
	}
}

func BenchmarkVerify(b *testing.B) {
	binder := New()
	binder.Bind("tok_prefix", "fp_abc123")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		binder.Verify("tok_prefix", "fp_abc123")
	}
}
