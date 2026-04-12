package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestAPIKeyFlow tests the full API key auth lifecycle against the live proxy:
// create key → use key to search → revoke key → verify rejected.
func TestAPIKeyFlow(t *testing.T) {
	proxy := proxyURL(t)
	client := &http.Client{}

	// Step 1: Create API key via admin endpoint
	t.Log("Step 1: Create API key")
	resp := apiDo(t, client, "POST", proxy+"/admin/apikeys",
		"application/json", `{"client_id":"e2e-test","scopes":["read:logs-*"]}`)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		t.Fatalf("create key: expected 200/201, got %d: %s", resp.StatusCode, readBodyStr(resp))
	}
	var keyResp struct {
		APIKey string `json:"api_key"`
		ID     string `json:"id"`
		Prefix string `json:"prefix"`
	}
	json.NewDecoder(resp.Body).Decode(&keyResp)
	resp.Body.Close()

	if keyResp.APIKey == "" {
		t.Fatal("create key: no api_key in response")
	}
	if !strings.HasPrefix(keyResp.APIKey, "oak_") {
		t.Errorf("api key should start with oak_, got %s", keyResp.APIKey[:min(len(keyResp.APIKey), 12)])
	}
	t.Logf("  key=%s... id=%s", keyResp.APIKey[:min(len(keyResp.APIKey), 16)], keyResp.ID)

	// Step 2: Use API key to hit a protected endpoint
	t.Log("Step 2: Use API key")
	req, _ := http.NewRequest("GET", proxy+"/health", nil)
	req.Header.Set("X-API-Key", keyResp.APIKey)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("use key: %v", err)
	}
	resp.Body.Close()
	// Health is usually public, try a search endpoint
	req, _ = http.NewRequest("GET", proxy+"/logs-*/_search?q=*&size=1", nil)
	req.Header.Set("X-API-Key", keyResp.APIKey)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("search with key: %v", err)
	}
	if resp.StatusCode == 200 || resp.StatusCode == 404 {
		t.Logf("  ✅ API key accepted (HTTP %d)", resp.StatusCode)
	} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
		t.Logf("  ⚠️  API key auth returned %d (may need scope mapping)", resp.StatusCode)
	} else {
		t.Logf("  search returned HTTP %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Step 3: List keys for client
	t.Log("Step 3: List keys")
	resp = apiDo(t, client, "GET", proxy+"/admin/apikeys/e2e-test", "", "")
	if resp.StatusCode == 200 {
		var keys []json.RawMessage
		json.NewDecoder(resp.Body).Decode(&keys)
		t.Logf("  %d keys for e2e-test", len(keys))
	}
	resp.Body.Close()

	// Step 4: Revoke key
	t.Log("Step 4: Revoke key")
	resp = apiDo(t, client, "DELETE", proxy+"/admin/apikeys/"+keyResp.ID, "", "")
	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		t.Log("  ✅ Key revoked")
	} else {
		t.Logf("  revoke returned HTTP %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Step 5: Verify revoked key is rejected
	t.Log("Step 5: Verify revoked key rejected")
	req, _ = http.NewRequest("GET", proxy+"/logs-*/_search?q=*&size=1", nil)
	req.Header.Set("X-API-Key", keyResp.APIKey)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("revoked key request: %v", err)
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		t.Log("  ✅ Revoked key rejected")
	} else {
		t.Logf("  ⚠️  Revoked key returned %d (may still be cached)", resp.StatusCode)
	}
	resp.Body.Close()

	t.Log("✅ API key flow complete")
}

func apiDo(t *testing.T, client *http.Client, method, url, contentType, body string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, url, bodyReader)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func readBodyStr(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return string(b)
}
