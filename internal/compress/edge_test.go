package compress

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Edge: large response gets compressed
func TestEdge_LargeResponseCompressed(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(strings.Repeat(`{"key":"value"},`, 1000)))
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("large response should be gzip compressed")
	}
	if w.Body.Len() >= 16000 {
		t.Errorf("compressed body should be smaller than 16000, got %d", w.Body.Len())
	}
}

// Edge: small response without Accept-Encoding not compressed
func TestEdge_NoAcceptEncodingNoCompress(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not compress without Accept-Encoding")
	}
}

// Edge: HEAD request passes through
func TestEdge_HeadPassthrough(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	r := httptest.NewRequest("HEAD", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("HEAD should pass through, got %d", w.Code)
	}
}

// Edge: response body readable after compression
func TestEdge_CompressedBodyReadable(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, strings.Repeat("test data ", 500))
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Body.Len() == 0 {
		t.Error("compressed body should not be empty")
	}
}
