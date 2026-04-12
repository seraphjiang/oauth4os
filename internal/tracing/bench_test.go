package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkSplitTraceparent_Valid(b *testing.B) {
	tp := "00-aaaabbbbccccddddeeee111122223333-1234567890abcdef-01"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitTraceparent(tp)
	}
}

func BenchmarkSplitTraceparent_Invalid(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitTraceparent("garbage")
	}
}

func BenchmarkMiddleware_WithTraceparent(b *testing.B) {
	tracer := NoopTracer{}
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), tracer)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("traceparent", "00-aaaabbbbccccddddeeee111122223333-1234567890abcdef-01")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func BenchmarkMiddleware_NoTraceparent(b *testing.B) {
	tracer := NoopTracer{}
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), tracer)
	req := httptest.NewRequest("GET", "/test", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func BenchmarkStartSpan(b *testing.B) {
	tracer := NoopTracer{}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := tracer.StartSpan(ctx, "test", nil)
		tracer.EndSpan(span, "ok")
	}
}
