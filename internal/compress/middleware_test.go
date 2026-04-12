package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipWhenAccepted(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("hello world ", 100)))
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("expected gzip content-encoding")
	}
	gr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	body, _ := io.ReadAll(gr)
	if !strings.Contains(string(body), "hello world") {
		t.Fatal("expected decompressed body to contain 'hello world'")
	}
}

func TestNoGzipWithoutHeader(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain"))
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("should not gzip without Accept-Encoding")
	}
	if rec.Body.String() != "plain" {
		t.Fatalf("expected 'plain', got %q", rec.Body.String())
	}
}
