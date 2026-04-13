package chaos

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkDisabled(b *testing.B) {
	inj := New(Config{ErrorRate: 1.0})
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := inj.Middleware(inner)
	r := httptest.NewRequest("GET", "/", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkEnabled_NoFault(b *testing.B) {
	inj := New(Config{ErrorRate: 0, LatencyRate: 0, DropRate: 0})
	inj.Enable()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := inj.Middleware(inner)
	r := httptest.NewRequest("GET", "/", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}
