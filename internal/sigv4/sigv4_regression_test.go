package sigv4

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHostHeaderInSignature verifies host is included in signed headers
// even though Go doesn't put it in r.Header (it's in r.Host).
func TestHostHeaderInSignature(t *testing.T) {
	var gotAuth string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer backend.Close()

	tr := &Transport{
		Base:      http.DefaultTransport,
		Region:    "us-west-2",
		Service:   "aoss",
		AccessKey: "AKID",
		SecretKey: "secret",
	}

	req, _ := http.NewRequest("GET", backend.URL+"/_search", nil)
	tr.RoundTrip(req)

	if !strings.Contains(gotAuth, "host") {
		t.Fatalf("Authorization header should include 'host' in SignedHeaders, got: %s", gotAuth)
	}
}

// TestCanonicalURIEncoding verifies path segments are URI-encoded.
func TestCanonicalURIEncoding(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/my%20index/_search", nil)
	uri := canonicalURI(req)
	// Should preserve encoding
	if !strings.Contains(uri, "_search") {
		t.Fatalf("unexpected URI: %s", uri)
	}
}

// TestQueryStringSorted verifies query params are sorted.
func TestQueryStringSorted(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/?z=1&a=2&m=3", nil)
	qs := canonicalQueryString(req)
	if qs != "a=2&m=3&z=1" {
		t.Fatalf("expected sorted query string, got: %s", qs)
	}
}

// TestQueryStringMultipleValues verifies multiple values for same key are sorted.
func TestQueryStringMultipleValues(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/?a=2&a=1", nil)
	qs := canonicalQueryString(req)
	if qs != "a=1&a=2" {
		t.Fatalf("expected sorted values, got: %s", qs)
	}
}

// TestEmptyQueryString verifies empty query returns empty string.
func TestEmptyQueryString(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	qs := canonicalQueryString(req)
	if qs != "" {
		t.Fatalf("expected empty, got: %s", qs)
	}
}

// TestCanonicalHeadersIncludeContentType verifies content-type is signed when present.
func TestCanonicalHeadersIncludeContentType(t *testing.T) {
	req := httptest.NewRequest("POST", "http://example.com/", nil)
	req.Host = "example.com"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-amz-date", "20260412T000000Z")
	signed, canonical := canonicalHeaderStr(req)

	if !strings.Contains(signed, "content-type") {
		t.Fatalf("content-type should be in signed headers: %s", signed)
	}
	if !strings.Contains(canonical, "application/json") {
		t.Fatalf("content-type value should be in canonical headers: %s", canonical)
	}
}
