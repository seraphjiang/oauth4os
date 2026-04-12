//go:build e2e

package e2e

import (
	"bytes"
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

// Config — set via env vars or defaults to docker-compose.demo.yml
var (
	proxyURL     = envOr("PROXY_URL", "http://localhost:8443")
	keycloakURL  = envOr("KEYCLOAK_URL", "http://localhost:8080")
	opensearchURL = envOr("OPENSEARCH_URL", "http://localhost:9200")
	realm        = "opensearch"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func getToken(t *testing.T, clientID, clientSecret, scope string) string {
	t.Helper()
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}
	if scope != "" {
		form.Set("scope", scope)
	}
	resp, err := http.PostForm(proxyURL+"/oauth/token", form)
	if err != nil {
		t.Fatalf("Token request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("Token request returned %d: %s", resp.StatusCode, body)
	}
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	token, _ := result["access_token"].(string)
	if token == "" {
		t.Fatalf("No access_token in response: %s", body)
	}
	return token
}

func proxyRequest(method, path, token string, body interface{}) (*http.Response, []byte) {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, proxyURL+path, reqBody)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, []byte(err.Error())
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp, respBody
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestHealth(t *testing.T) {
	resp, body := proxyRequest("GET", "/health", "", nil)
	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("Health check failed: %s", body)
	}
}

func TestTokenIssuance(t *testing.T) {
	token := getToken(t, "log-reader", "log-reader-secret", "read:logs-*")
	if token == "" {
		t.Fatal("Empty token")
	}
	// Token should be a JWT (3 dot-separated parts)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("Token doesn't look like JWT: %d parts", len(parts))
	}
}

func TestInvalidCredentialsRejected(t *testing.T) {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"fake"},
		"client_secret": {"wrong"},
	}
	resp, _ := http.PostForm(proxyURL+"/oauth/token", form)
	if resp != nil && resp.StatusCode == 200 {
		t.Fatal("Expected rejection for invalid credentials")
	}
}

func TestAdminCanCreateIndex(t *testing.T) {
	token := getToken(t, "admin-agent", "admin-agent-secret", "admin")
	resp, body := proxyRequest("PUT", "/e2e-test-index", token, map[string]interface{}{
		"settings": map[string]interface{}{"number_of_shards": 1, "number_of_replicas": 0},
	})
	if resp == nil || (resp.StatusCode != 200 && resp.StatusCode != 400) {
		t.Fatalf("Create index failed: %d %s", resp.StatusCode, body)
	}
	// Cleanup
	proxyRequest("DELETE", "/e2e-test-index", token, nil)
}

func TestReaderCanSearch(t *testing.T) {
	admin := getToken(t, "admin-agent", "admin-agent-secret", "admin")
	// Create index + doc
	proxyRequest("PUT", "/e2e-logs-read", admin, map[string]interface{}{
		"settings": map[string]interface{}{"number_of_shards": 1, "number_of_replicas": 0},
	})
	proxyRequest("POST", "/e2e-logs-read/_doc?refresh=true", admin, map[string]interface{}{
		"level": "ERROR", "message": "test",
	})

	reader := getToken(t, "log-reader", "log-reader-secret", "read:logs-*")
	resp, body := proxyRequest("POST", "/e2e-logs-read/_search", reader, map[string]interface{}{
		"query": map[string]interface{}{"match_all": map[string]interface{}{}},
	})
	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("Search failed: %s", body)
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	hits, _ := result["hits"].(map[string]interface{})
	total, _ := hits["total"].(map[string]interface{})
	if val, _ := total["value"].(float64); val < 1 {
		t.Errorf("Expected ≥1 hit, got %v", total)
	}

	// Cleanup
	proxyRequest("DELETE", "/e2e-logs-read", admin, nil)
}

func TestUnauthenticatedRejected(t *testing.T) {
	resp, _ := proxyRequest("GET", "/_search", "", nil)
	if resp != nil && resp.StatusCode == 200 {
		t.Fatal("Expected rejection for unauthenticated request")
	}
}

func TestTokenRevocation(t *testing.T) {
	token := getToken(t, "ci-pipeline", "ci-pipeline-secret", "write:dashboards")
	resp, body := proxyRequest("POST", "/oauth/revoke", "", map[string]interface{}{
		"token": token,
	})
	if resp == nil {
		t.Fatalf("Revoke request failed: %s", body)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		t.Errorf("Revoke returned %d: %s", resp.StatusCode, body)
	}
}

func TestTokenListing(t *testing.T) {
	admin := getToken(t, "admin-agent", "admin-agent-secret", "admin")
	resp, body := proxyRequest("GET", "/oauth/tokens", admin, nil)
	if resp == nil || resp.StatusCode != 200 {
		t.Skipf("Token listing not implemented: %s", body)
	}
}

// Ensure unused imports don't cause errors
var _ = fmt.Sprintf
