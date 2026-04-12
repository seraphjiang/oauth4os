package sigv4

import (
	"net/http"
	"strings"
	"testing"
)

// FuzzCanonicalURI ensures canonicalURI never panics on arbitrary paths.
func FuzzCanonicalURI(f *testing.F) {
	f.Add("/")
	f.Add("/logs-demo/_search")
	f.Add("/a/b/c/../d")
	f.Add("/%2F%2F")
	f.Add("/foo bar/baz")
	f.Add("")
	f.Fuzz(func(t *testing.T, path string) {
		req, err := http.NewRequest("GET", "http://example.com"+path, nil)
		if err != nil {
			return // invalid URL is fine
		}
		result := canonicalURI(req)
		if !strings.HasPrefix(result, "/") {
			t.Errorf("canonical URI must start with /, got %q", result)
		}
	})
}

// FuzzCanonicalQueryString ensures query string canonicalization never panics.
func FuzzCanonicalQueryString(f *testing.F) {
	f.Add("")
	f.Add("a=1&b=2")
	f.Add("z=3&a=1&m=2")
	f.Add("key=%20value&foo=bar%26baz")
	f.Add("=empty&novalue&a=1")
	f.Fuzz(func(t *testing.T, qs string) {
		req, err := http.NewRequest("GET", "http://example.com/?"+qs, nil)
		if err != nil {
			return
		}
		_ = canonicalQueryString(req)
	})
}

// FuzzURIEncode ensures uriEncode never panics.
func FuzzURIEncode(f *testing.F) {
	f.Add("hello world", true)
	f.Add("/path/to/resource", false)
	f.Add("special!@#$%^&*()", true)
	f.Add("", false)
	f.Add("日本語", true)
	f.Fuzz(func(t *testing.T, s string, encodeSlash bool) {
		result := uriEncode(s, encodeSlash)
		if encodeSlash && strings.Contains(s, "/") && strings.Contains(result, "/") {
			t.Errorf("slash should be encoded when encodeSlash=true")
		}
	})
}
