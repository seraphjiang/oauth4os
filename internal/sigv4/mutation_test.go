package sigv4

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mutation: remove Authorization header → signed request must have Authorization
func TestMutation_AuthorizationHeader(t *testing.T) {
	var gotAuth string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer backend.Close()
	tr := New(http.DefaultTransport, "us-east-1", "es")
	req, _ := http.NewRequest("GET", backend.URL+"/test", nil)
	tr.RoundTrip(req)
	if gotAuth == "" {
		t.Error("signed request must include Authorization header")
	}
}

// Mutation: remove canonical URI → must produce valid canonical URI
func TestMutation_CanonicalURI(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://example.com/logs-2024/_search", nil)
	uri := canonicalURI(r)
	if uri != "/logs-2024/_search" {
		t.Errorf("expected /logs-2024/_search, got %s", uri)
	}
}

// Mutation: remove root path handling → root must return /
func TestMutation_CanonicalURIRoot(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://example.com/", nil)
	uri := canonicalURI(r)
	if uri != "/" {
		t.Errorf("root path should be /, got %s", uri)
	}
}
