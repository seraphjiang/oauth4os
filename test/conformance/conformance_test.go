//go:build conformance

// API conformance tests — verify all OpenAPI spec endpoints exist and return
// expected status codes and content types.
//
// Run: docker compose up -d && go test ./test/conformance/ -v -count=1

package conformance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

var proxyURL = getEnv("PROXY_URL", "http://localhost:8443")

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func waitForProxy(t *testing.T) {
	t.Helper()
	for i := 0; i < 30; i++ {
		resp, err := http.Get(proxyURL + "/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("proxy not ready after 30s")
}

type endpoint struct {
	method      string
	path        string
	body        string // form-encoded or JSON
	contentType string
	wantStatus  int
	wantCT      string // expected Content-Type prefix
}

func TestConformance_AllEndpoints(t *testing.T) {
	waitForProxy(t)

	// Issue a token for authenticated endpoints
	token := issueTestToken(t)

	endpoints := []endpoint{
		// Operational
		{"GET", "/health", "", "", 200, "application/json"},
		{"GET", "/metrics", "", "", 200, "text/plain"},

		// Discovery
		{"GET", "/.well-known/openid-configuration", "", "", 200, "application/json"},

		// Token lifecycle
		{"POST", "/oauth/token", "grant_type=client_credentials&client_id=conformance&client_secret=test&scope=read:logs-*", "application/x-www-form-urlencoded", 200, "application/json"},
		{"GET", "/oauth/tokens", "", "", 200, "application/json"},
		{"GET", "/oauth/token/" + token, "", "", 200, "application/json"},
		{"DELETE", "/oauth/token/" + token, "", "", 204, ""},

		// Token errors
		{"POST", "/oauth/token", "grant_type=bad", "application/x-www-form-urlencoded", 400, ""},

		// Introspection
		{"POST", "/oauth/introspect", "token=nonexistent", "application/x-www-form-urlencoded", 200, "application/json"},

		// Client registration
		{"POST", "/oauth/register", `{"client_name":"conformance-test"}`, "application/json", 201, "application/json"},

		// Admin API
		{"GET", "/admin/scope-mappings", "", "", 200, "application/json"},
		{"GET", "/admin/providers", "", "", 200, "application/json"},
		{"GET", "/admin/tenants", "", "", 200, "application/json"},
		{"GET", "/admin/cedar-policies", "", "", 200, "application/json"},
		{"GET", "/admin/rate-limits", "", "", 200, "application/json"},
		{"GET", "/admin/config", "", "", 200, "application/json"},

		// Proxy passthrough (OpenSearch root)
		{"GET", "/", "", "", 0, ""}, // any status — just verify proxy responds

		// Invalid bearer → 401
		{"GET", "/_search", "", "", 401, ""},
	}

	for _, ep := range endpoints {
		name := fmt.Sprintf("%s %s → %d", ep.method, ep.path, ep.wantStatus)
		t.Run(name, func(t *testing.T) {
			var bodyReader io.Reader
			if ep.body != "" {
				bodyReader = strings.NewReader(ep.body)
			}
			req, err := http.NewRequest(ep.method, proxyURL+ep.path, bodyReader)
			if err != nil {
				t.Fatalf("request build failed: %v", err)
			}
			if ep.contentType != "" {
				req.Header.Set("Content-Type", ep.contentType)
			}
			// Add invalid bearer for the 401 test
			if ep.path == "/_search" {
				req.Header.Set("Authorization", "Bearer invalid.jwt.token")
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if ep.wantStatus != 0 && resp.StatusCode != ep.wantStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected %d, got %d: %s", ep.wantStatus, resp.StatusCode, body)
			}

			if ep.wantCT != "" {
				ct := resp.Header.Get("Content-Type")
				if !strings.HasPrefix(ct, ep.wantCT) {
					t.Fatalf("expected Content-Type %s, got %s", ep.wantCT, ct)
				}
			}
		})
	}
}

func TestConformance_OIDCDiscoveryFields(t *testing.T) {
	waitForProxy(t)
	resp, err := http.Get(proxyURL + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var doc map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&doc)

	required := []string{"issuer", "token_endpoint", "introspection_endpoint", "grant_types_supported"}
	for _, field := range required {
		if doc[field] == nil {
			t.Errorf("missing required OIDC field: %s", field)
		}
	}
}

func TestConformance_TokenResponseFields(t *testing.T) {
	waitForProxy(t)
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"field-check"},
		"client_secret": {"test"},
		"scope":         {"read:logs-*"},
	}
	resp, err := http.PostForm(proxyURL+"/oauth/token", data)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var tok map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tok)

	required := []string{"access_token", "token_type", "expires_in"}
	for _, field := range required {
		if tok[field] == nil {
			t.Errorf("missing required token field: %s", field)
		}
	}
	if tok["token_type"] != "Bearer" {
		t.Errorf("expected token_type Bearer, got %v", tok["token_type"])
	}
}

func TestConformance_ErrorResponseFormat(t *testing.T) {
	waitForProxy(t)
	data := url.Values{"grant_type": {"invalid_type"}}
	resp, err := http.PostForm(proxyURL+"/oauth/token", data)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var errResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errResp)

	if errResp["error"] == nil {
		t.Error("error response missing 'error' field")
	}
}

func issueTestToken(t *testing.T) string {
	t.Helper()
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"conformance-token"},
		"client_secret": {"test"},
		"scope":         {"read:logs-*"},
	}
	resp, err := http.PostForm(proxyURL+"/oauth/token", data)
	if err != nil {
		t.Fatalf("token issuance failed: %v", err)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	tok, _ := result["access_token"].(string)
	if tok == "" {
		t.Fatal("no access_token")
	}
	return tok
}

func TestMain(m *testing.M) {
	fmt.Println("oauth4os API conformance tests")
	os.Exit(m.Run())
}
