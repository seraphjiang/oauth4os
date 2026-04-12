package sigv4

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkCanonicalQueryString(b *testing.B) {
	r := httptest.NewRequest("GET", "http://example.com/?z=1&a=2&m=3&q=hello+world", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		canonicalQueryString(r)
	}
}

func BenchmarkCanonicalHeaders(b *testing.B) {
	r := httptest.NewRequest("POST", "http://example.com/", nil)
	r.Host = "example.com"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("x-amz-date", "20260412T000000Z")
	r.Header.Set("x-amz-security-token", "token123")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		canonicalHeaderStr(r)
	}
}

func BenchmarkURIEncode(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uriEncode("/my index/_search?q=hello world", false)
	}
}
