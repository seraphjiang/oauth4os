package etag

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func BenchmarkETag_SmallBody(b *testing.B) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	}))
	req := httptest.NewRequest("GET", "/", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func BenchmarkETag_LargeBody(b *testing.B) {
	body := []byte(strings.Repeat(`{"level":"INFO","service":"payment","msg":"ok"}`, 100))
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func BenchmarkETag_304(b *testing.B) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("stable content"))
	}))
	// Get the ETag first
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	tag := rec.Header().Get("ETag")

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("If-None-Match", tag)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
}
