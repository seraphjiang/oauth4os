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

func TestTransport_SignsBody(t *testing.T) {
	var gotContentHash string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentHash = r.Header.Get("x-amz-content-sha256")
		w.WriteHeader(200)
	}))
	defer backend.Close()

	tr := &Transport{
		Base: http.DefaultTransport, Region: "us-west-2", Service: "aoss",
		AccessKey: "AKID", SecretKey: "SECRET",
	}

	req, _ := http.NewRequest("POST", backend.URL+"/_search", strings.NewReader(`{"query":{}}`))
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotContentHash == "" {
		t.Error("x-amz-content-sha256 missing")
	}
	// Empty body hash is e3b0c44... — this should be different
	if gotContentHash == "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Error("body hash is empty-body hash, but we sent a body")
	}
}

func TestTransport_SessionToken(t *testing.T) {
	var gotToken string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("x-amz-security-token")
		w.WriteHeader(200)
	}))
	defer backend.Close()

	tr := &Transport{
		Base: http.DefaultTransport, Region: "us-west-2", Service: "aoss",
		AccessKey: "AKID", SecretKey: "SECRET", Token: "SESSION_TOKEN",
	}

	req, _ := http.NewRequest("GET", backend.URL+"/", nil)
	resp, _ := tr.RoundTrip(req)
	resp.Body.Close()
	if gotToken != "SESSION_TOKEN" {
		t.Errorf("session token = %q, want SESSION_TOKEN", gotToken)
	}
}

func TestTransport_NoSessionToken(t *testing.T) {
	var gotToken string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("x-amz-security-token")
		w.WriteHeader(200)
	}))
	defer backend.Close()

	tr := &Transport{
		Base: http.DefaultTransport, Region: "us-west-2", Service: "aoss",
		AccessKey: "AKID", SecretKey: "SECRET",
	}

	req, _ := http.NewRequest("GET", backend.URL+"/", nil)
	resp, _ := tr.RoundTrip(req)
	resp.Body.Close()
	if gotToken != "" {
		t.Errorf("session token should be empty, got %q", gotToken)
	}
}

func TestCanonicalQueryString(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com/index/_search?size=10&q=level:ERROR&sort=@timestamp", nil)
	qs := canonicalQueryString(req)
	// Must be sorted alphabetically
	if !strings.HasPrefix(qs, "q=") {
		t.Errorf("query string not sorted: %s", qs)
	}
	if !strings.Contains(qs, "size=10") {
		t.Errorf("missing size param: %s", qs)
	}
}

func TestURIEncode(t *testing.T) {
	tests := []struct{ in, want string }{
		{"simple", "simple"},
		{"hello world", "hello%20world"},
		{"a/b", "a/b"},
		{"a+b", "a%2Bb"},
	}
	for _, tt := range tests {
		got := uriEncode(tt.in, false)
		if got != tt.want {
			t.Errorf("uriEncode(%q, false) = %q, want %q", tt.in, got, tt.want)
		}
	}
	// With encodeSlash
	if got := uriEncode("a/b", true); got != "a%2Fb" {
		t.Errorf("uriEncode(a/b, true) = %q, want a%%2Fb", got)
	}
}

func TestJsonVal(t *testing.T) {
	body := `{"AccessKeyId":"AKID","SecretAccessKey":"SECRET","Token":"TOK","Expiration":"2026-01-01T00:00:00Z"}`
	if v := jsonVal(body, "AccessKeyId"); v != "AKID" {
		t.Errorf("AccessKeyId = %q", v)
	}
	if v := jsonVal(body, "Token"); v != "TOK" {
		t.Errorf("Token = %q", v)
	}
	if v := jsonVal(body, "Missing"); v != "" {
		t.Errorf("Missing = %q, want empty", v)
	}
}

func TestNew_WithEnvVars(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AKID")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "SECRET")
	t.Setenv("AWS_SESSION_TOKEN", "TOK")
	tr := New(nil, "us-west-2", "es")
	if tr.AccessKey != "AKID" {
		t.Errorf("AccessKey = %q", tr.AccessKey)
	}
	if tr.SecretKey != "SECRET" {
		t.Errorf("SecretKey = %q", tr.SecretKey)
	}
	if tr.Token != "TOK" {
		t.Errorf("Token = %q", tr.Token)
	}
	if tr.Region != "us-west-2" {
		t.Errorf("Region = %q", tr.Region)
	}
}

func TestNew_FallbackToContainerCreds(t *testing.T) {
	// Clear env vars so it tries container creds
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	// No container URI set — should still return a transport
	t.Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "")
	tr := New(nil, "us-east-1", "aoss")
	if tr == nil {
		t.Fatal("New returned nil")
	}
	if tr.Region != "us-east-1" {
		t.Errorf("Region = %q", tr.Region)
	}
}

func TestCanonicalHeaderStr_MultiValue(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://search.us-west-2.es.amazonaws.com/", nil)
	req.Header.Set("X-Amz-Date", "20260412T190000Z")
	req.Header.Set("Host", "search.us-west-2.es.amazonaws.com")
	signed, canonical := canonicalHeaderStr(req)
	if signed == "" || canonical == "" {
		t.Error("expected non-empty canonical headers")
	}
}
