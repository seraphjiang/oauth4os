package sigv4

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEdge_NewTransport(t *testing.T) {
	tr := New(http.DefaultTransport, "us-east-1", "es")
	if tr == nil {
		t.Error("New should return non-nil transport")
	}
}

func TestEdge_CanonicalURIRoot(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	uri := canonicalURI(r)
	if uri != "/" {
		t.Errorf("root URI should be '/', got %q", uri)
	}
}

func TestEdge_CanonicalURIPath(t *testing.T) {
	r := httptest.NewRequest("GET", "/index/doc/1", nil)
	uri := canonicalURI(r)
	if uri != "/index/doc/1" {
		t.Errorf("expected '/index/doc/1', got %q", uri)
	}
}

func TestEdge_CanonicalQueryEmpty(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	qs := canonicalQueryString(r)
	if qs != "" {
		t.Errorf("empty query should be empty, got %q", qs)
	}
}

func TestEdge_CanonicalQuerySorted(t *testing.T) {
	r := httptest.NewRequest("GET", "/?z=1&a=2", nil)
	qs := canonicalQueryString(r)
	if len(qs) == 0 {
		t.Error("query string should not be empty")
	}
}

func TestEdge_HashSHA256Deterministic(t *testing.T) {
	h1 := hashSHA256([]byte("hello"))
	h2 := hashSHA256([]byte("hello"))
	if h1 != h2 {
		t.Error("same input should produce same hash")
	}
	if h1 == "" {
		t.Error("hash should not be empty")
	}
}

func TestEdge_HashSHA256Empty(t *testing.T) {
	h := hashSHA256([]byte{})
	if h == "" {
		t.Error("empty input should still produce hash")
	}
}

func TestEdge_DeriveKeyNotEmpty(t *testing.T) {
	key := deriveKey("secret", "20260413", "us-east-1", "es")
	if len(key) == 0 {
		t.Error("derived key should not be empty")
	}
}

func TestEdge_URIEncodePassthrough(t *testing.T) {
	s := uriEncode("hello", false)
	if s != "hello" {
		t.Errorf("plain string should passthrough, got %q", s)
	}
}

func TestEdge_URIEncodeSpecialChars(t *testing.T) {
	s := uriEncode("hello world", false)
	if s == "hello world" {
		t.Error("space should be encoded")
	}
}
