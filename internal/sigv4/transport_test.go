package sigv4

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTransport_SetsHostAndAuthHeaders(t *testing.T) {
	// Capture what the transport sends to the backend
	var gotHost, gotAuth, gotAmzDate string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
		gotAuth = r.Header.Get("Authorization")
		gotAmzDate = r.Header.Get("x-amz-date")
		w.WriteHeader(200)
	}))
	defer backend.Close()

	tr := &Transport{
		Base:      http.DefaultTransport,
		Region:    "us-west-2",
		Service:   "aoss",
		AccessKey: "AKIAIOSFODNN7EXAMPLE",
		SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	req, _ := http.NewRequest("GET", backend.URL+"/logs-demo/_search", nil)
	// Simulate reverse proxy: Host is the proxy, URL is the backend
	req.Host = "my-proxy.example.com"

	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Host must be the backend, not the proxy
	backendHost := strings.TrimPrefix(backend.URL, "http://")
	if gotHost != backendHost {
		t.Errorf("Host = %q, want %q", gotHost, backendHost)
	}
	if !strings.HasPrefix(gotAuth, "AWS4-HMAC-SHA256 Credential=AKIA") {
		t.Errorf("Authorization missing or wrong: %q", gotAuth)
	}
	if gotAmzDate == "" {
		t.Error("x-amz-date header missing")
	}
}

func TestTransport_StripsProxyHeaders(t *testing.T) {
	var gotAuthHeader string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer backend.Close()

	tr := &Transport{
		Base:      http.DefaultTransport,
		Region:    "us-west-2",
		Service:   "aoss",
		AccessKey: "AKIAIOSFODNN7EXAMPLE",
		SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	req, _ := http.NewRequest("GET", backend.URL+"/test", nil)
	req.Header.Set("Authorization", "Bearer old-jwt-token")

	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Must be SigV4, not the original Bearer token
	if strings.Contains(gotAuthHeader, "Bearer") {
		t.Error("Bearer token leaked through to upstream")
	}
	if !strings.HasPrefix(gotAuthHeader, "AWS4-HMAC-SHA256") {
		t.Errorf("expected SigV4 auth, got: %q", gotAuthHeader)
	}
}
