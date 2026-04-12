package sigv4

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Property: every signed request has a valid Authorization header format.
func TestProperty_AuthHeaderFormat(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 Credential=") {
			t.Errorf("bad auth prefix: %q", auth)
		}
		if !strings.Contains(auth, "SignedHeaders=") {
			t.Errorf("missing SignedHeaders: %q", auth)
		}
		if !strings.Contains(auth, "Signature=") {
			t.Errorf("missing Signature: %q", auth)
		}
		w.WriteHeader(200)
	}))
	defer backend.Close()

	tr := testTransport()
	methods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "PATCH"}
	paths := []string{"/", "/a", "/logs-demo/_search", "/a/b/c/d", "/_cat/indices"}

	for i := 0; i < 50; i++ {
		method := methods[rand.Intn(len(methods))]
		path := paths[rand.Intn(len(paths))]
		req, _ := http.NewRequest(method, backend.URL+path, nil)
		resp, err := tr.RoundTrip(req)
		if err != nil {
			t.Fatalf("request %d (%s %s): %v", i, method, path, err)
		}
		resp.Body.Close()
	}
}

// Property: x-amz-date is always present and in ISO 8601 basic format.
func TestProperty_AmzDatePresent(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := r.Header.Get("x-amz-date")
		if len(d) != 16 || d[8] != 'T' || d[15] != 'Z' {
			t.Errorf("bad x-amz-date format: %q", d)
		}
		w.WriteHeader(200)
	}))
	defer backend.Close()

	tr := testTransport()
	for i := 0; i < 20; i++ {
		req, _ := http.NewRequest("GET", backend.URL+"/test", nil)
		resp, _ := tr.RoundTrip(req)
		resp.Body.Close()
	}
}

// Property: host header always matches backend, never the proxy.
func TestProperty_HostMatchesBackend(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Host, "evil") || strings.Contains(r.Host, "proxy") {
			t.Errorf("host leaked proxy value: %q", r.Host)
		}
		w.WriteHeader(200)
	}))
	defer backend.Close()

	tr := testTransport()
	proxyHosts := []string{"proxy.example.com", "evil.com", "localhost:9999"}
	for _, ph := range proxyHosts {
		req, _ := http.NewRequest("GET", backend.URL+"/test", nil)
		req.Host = ph
		resp, _ := tr.RoundTrip(req)
		resp.Body.Close()
	}
}

// Property: original Bearer token never reaches upstream.
func TestProperty_BearerNeverLeaks(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if strings.Contains(auth, "Bearer") {
			t.Errorf("Bearer token leaked: %q", auth)
		}
		w.WriteHeader(200)
	}))
	defer backend.Close()

	tr := testTransport()
	tokens := []string{"Bearer tok_abc", "Bearer eyJhbGciOiJSUzI1NiJ9.test.sig", "bearer lowercase"}
	for _, tok := range tokens {
		req, _ := http.NewRequest("GET", backend.URL+"/test", nil)
		req.Header.Set("Authorization", tok)
		resp, _ := tr.RoundTrip(req)
		resp.Body.Close()
	}
}

// Property: signature changes when path changes.
func TestProperty_SignatureVariesByPath(t *testing.T) {
	sigs := map[string]string{}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		sig := auth[strings.LastIndex(auth, "Signature=")+10:]
		sigs[r.URL.Path] = sig
		w.WriteHeader(200)
	}))
	defer backend.Close()

	tr := testTransport()
	paths := []string{"/a", "/b", "/c"}
	for _, p := range paths {
		req, _ := http.NewRequest("GET", backend.URL+p, nil)
		resp, _ := tr.RoundTrip(req)
		resp.Body.Close()
	}
	// At least 2 distinct signatures for 3 different paths
	unique := map[string]bool{}
	for _, s := range sigs {
		unique[s] = true
	}
	if len(unique) < 2 {
		t.Errorf("expected different signatures for different paths, got %d unique", len(unique))
	}
}

// Property: query string ordering doesn't break signing.
func TestProperty_QueryStringOrder(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256") {
			t.Errorf("signing failed with query: %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
	}))
	defer backend.Close()

	tr := testTransport()
	queries := []string{
		"?a=1&b=2&c=3",
		"?c=3&b=2&a=1",
		"?format=json&size=10",
		"?q=level:ERROR&size=5&from=0",
		"",
	}
	for _, q := range queries {
		req, _ := http.NewRequest("GET", backend.URL+"/test"+q, nil)
		resp, _ := tr.RoundTrip(req)
		resp.Body.Close()
	}
}

func testTransport() *Transport {
	return &Transport{
		Base:      http.DefaultTransport,
		Region:    "us-west-2",
		Service:   "aoss",
		AccessKey: "AKIAIOSFODNN7EXAMPLE",
		SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
}
