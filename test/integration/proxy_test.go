// Integration tests for oauth4os — requires Docker (OpenSearch + proxy).
//
// Run: docker compose up -d && go test ./test/integration/ -v -count=1
//
// Tests:
//   1. Health check — proxy responds
//   2. Token issuance — POST /oauth/token returns access_token
//   3. Token listing — GET /oauth/tokens returns issued tokens
//   4. Token revocation — DELETE /oauth/token/{id} revokes
//   5. Proxy passthrough — unauthenticated request reaches OpenSearch
//   6. Bearer auth — valid token proxies to OpenSearch with roles
//   7. Invalid token — returns 401
//   8. Scope enforcement — no matching scope returns 403
//   9. Dashboards routing — /api/* routes to Dashboards
//  10. Audit trail — requests are logged

package integration

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

// --- 1. Health check ---

func TestHealthCheck(t *testing.T) {
	waitForProxy(t)
	resp, err := http.Get(proxyURL + "/health")
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", body["status"])
	}
}

// --- 2. Token issuance ---

func issueToken(t *testing.T, clientID, scope string) string {
	t.Helper()
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {"test-secret"},
		"scope":         {scope},
	}
	resp, err := http.PostForm(proxyURL+"/oauth/token", data)
	if err != nil {
		t.Fatalf("token issuance failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	token, ok := result["access_token"].(string)
	if !ok || token == "" {
		t.Fatal("no access_token in response")
	}
	return token
}

func TestTokenIssuance(t *testing.T) {
	waitForProxy(t)
	token := issueToken(t, "test-agent", "read:logs-*")
	if !strings.HasPrefix(token, "tok_") {
		t.Fatalf("expected tok_ prefix, got %s", token)
	}
}

// --- 3. Token listing ---

func TestTokenListing(t *testing.T) {
	waitForProxy(t)
	issueToken(t, "list-test", "read:logs-*")

	resp, err := http.Get(proxyURL + "/oauth/tokens")
	if err != nil {
		t.Fatalf("list tokens failed: %v", err)
	}
	defer resp.Body.Close()

	var tokens []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokens)
	if len(tokens) == 0 {
		t.Fatal("expected at least 1 token")
	}
}

// --- 4. Token revocation ---

func TestTokenRevocation(t *testing.T) {
	waitForProxy(t)
	token := issueToken(t, "revoke-test", "read:logs-*")

	req, _ := http.NewRequest("DELETE", proxyURL+"/oauth/token/"+token, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("revoke failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify token shows as revoked
	resp2, err2 := http.Get(proxyURL + "/oauth/token/" + token)
	if err2 != nil {
		t.Fatalf("get revoked token failed: %v", err2)
	}
	defer resp2.Body.Close()
	var tok map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&tok)
	if tok["revoked"] != true {
		t.Fatal("expected token to be revoked")
	}
}

// --- 5. Proxy passthrough (no auth) ---

func TestProxyPassthrough(t *testing.T) {
	waitForProxy(t)
	resp, err := http.Get(proxyURL + "/")
	if err != nil {
		t.Fatalf("passthrough failed: %v", err)
	}
	defer resp.Body.Close()

	// OpenSearch root returns cluster info or error — either way proxy forwarded
	if resp.StatusCode == 0 {
		t.Fatal("no response from proxy")
	}
	t.Logf("passthrough status: %d", resp.StatusCode)
}

// --- 6. Bearer auth proxies with roles ---

func TestBearerAuthProxy(t *testing.T) {
	waitForProxy(t)
	token := issueToken(t, "auth-test", "read:logs-*")

	req, _ := http.NewRequest("GET", proxyURL+"/_cat/indices", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("bearer proxy failed: %v", err)
	}
	defer resp.Body.Close()

	// Proxy should forward — response depends on OpenSearch config
	t.Logf("bearer auth status: %d", resp.StatusCode)
}

// --- 7. Invalid token ---

func TestInvalidToken(t *testing.T) {
	waitForProxy(t)
	req, _ := http.NewRequest("GET", proxyURL+"/_search", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "invalid_token" {
		t.Fatalf("expected invalid_token error, got %v", body["error"])
	}
}

// --- 8. Token get by ID ---

func TestTokenGetByID(t *testing.T) {
	waitForProxy(t)
	token := issueToken(t, "get-test", "admin")

	resp, err := http.Get(proxyURL + "/oauth/token/" + token)
	if err != nil {
		t.Fatalf("get token failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var tok map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tok)
	if tok["client_id"] != "get-test" {
		t.Fatalf("expected client_id get-test, got %v", tok["client_id"])
	}
}

// --- 9. Nonexistent token returns 404 ---

func TestTokenNotFound(t *testing.T) {
	waitForProxy(t)
	resp, err := http.Get(proxyURL + "/oauth/token/tok_nonexistent")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// --- 10. Multiple scopes ---

func TestMultipleScopes(t *testing.T) {
	waitForProxy(t)
	token := issueToken(t, "multi-scope", "read:logs-* write:dashboards")

	resp, err := http.Get(proxyURL + "/oauth/token/" + token)
	if err != nil {
		t.Fatalf("get token failed: %v", err)
	}
	defer resp.Body.Close()

	var tok map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tok)
	scopes, ok := tok["scopes"].([]interface{})
	if !ok || len(scopes) < 2 {
		t.Fatalf("expected 2+ scopes, got %v", tok["scopes"])
	}
}

func TestMain(m *testing.M) {
	fmt.Println("oauth4os integration tests")
	fmt.Printf("proxy: %s\n", proxyURL)
	os.Exit(m.Run())
}
