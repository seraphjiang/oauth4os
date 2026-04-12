package e2e

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestMetricsEndpoint verifies /metrics returns Prometheus-format metrics.
func TestMetricsEndpoint(t *testing.T) {
	proxy := proxyURL(t)
	resp, err := http.Get(proxy + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	text := string(body)

	required := []string{
		"oauth4os_requests_total",
		"oauth4os_auth_success",
		"oauth4os_auth_failed",
		"oauth4os_cedar_denied",
		"oauth4os_upstream_errors",
	}
	for _, m := range required {
		if !strings.Contains(text, m) {
			t.Errorf("missing metric: %s", m)
		}
	}

	// Verify TYPE annotations present
	if !strings.Contains(text, "# TYPE") {
		t.Error("missing TYPE annotations")
	}
}

// TestVersionEndpoint verifies /version returns version info.
func TestVersionEndpoint(t *testing.T) {
	proxy := proxyURL(t)
	resp, err := http.Get(proxy + "/version")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("empty version response")
	}
}

// TestOIDCDiscovery verifies the OIDC discovery document.
func TestOIDCDiscovery(t *testing.T) {
	proxy := proxyURL(t)
	resp, err := http.Get(proxy + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	text := string(body)
	for _, field := range []string{"issuer", "authorization_endpoint", "token_endpoint", "jwks_uri"} {
		if !strings.Contains(text, field) {
			t.Errorf("OIDC discovery missing field: %s", field)
		}
	}
}

// TestJWKSEndpoint verifies JWKS returns valid JSON with keys.
func TestJWKSEndpoint(t *testing.T) {
	proxy := proxyURL(t)
	resp, err := http.Get(proxy + "/.well-known/jwks.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "keys") {
		t.Error("JWKS missing 'keys' field")
	}
}
