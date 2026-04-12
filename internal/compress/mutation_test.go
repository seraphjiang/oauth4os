package compress

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mutation: remove gzip encoding → must compress when Accept-Encoding: gzip
func TestMutation_GzipResponse(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world hello world hello world"))
	})
	handler := Middleware(inner)
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("must set Content-Encoding: gzip")
	}
}

// Mutation: remove passthrough → must not compress without Accept-Encoding
func TestMutation_NoGzipWithout(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})
	handler := Middleware(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("must not compress without Accept-Encoding: gzip")
	}
	body, _ := io.ReadAll(w.Body)
	if string(body) != "hello" {
		t.Errorf("uncompressed body should be 'hello', got %q", body)
	}
}
