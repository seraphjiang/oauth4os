package dpop

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkValidate(b *testing.B) {
	r := httptest.NewRequest("POST", "/token", nil)
	r.Header.Set("DPoP", "eyJhbGciOiJFUzI1NiIsInR5cCI6ImRwb3Arand0IiwiandrIjp7Imt0eSI6IkVDIiwiY3J2IjoiUC0yNTYiLCJ4IjoiZjgzT0o1X0VzOTdQOTNDQm1mT2U2S0JRRkFNMVE5VjlrNjBBZFl4WGM0cyIsInkiOiJ4N3lITHg0X3hLZUhudTFsYWpGQnZfYWQ1ZDlpSUZNX1pDRHZGMmhfNjVJIn19.eyJqdGkiOiJ0ZXN0IiwiaHRtIjoiUE9TVCIsImh0dSI6Imh0dHBzOi8vZXhhbXBsZS5jb20vdG9rZW4iLCJpYXQiOjE3MDAwMDAwMDB9.invalid")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Validate(r) // will fail validation but measures parsing overhead
	}
}

func BenchmarkJWKThumbprint(b *testing.B) {
	jwk := []byte(`{"kty":"EC","crv":"P-256","x":"f83OJ5_Es97P93CBmfOe6KBQFam1Q9V9k60AdYxXc4s","y":"x7yHLx4_xKeHnu1lajFBv_ad5d9iIFM_ZCDvF2h_65I"}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JWKThumbprint(jwk)
	}
}
