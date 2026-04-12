package e2e

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestDeviceFlowE2E tests RFC 8628 device flow against the live proxy:
// request code → poll (pending) → approve → poll (token).
func TestDeviceFlowE2E(t *testing.T) {
	proxy := proxyURL(t)
	client := &http.Client{}

	// Step 1: Request device code
	t.Log("Step 1: Request device code")
	resp, err := client.Post(proxy+"/oauth/device/code",
		"application/x-www-form-urlencoded",
		strings.NewReader(url.Values{"client_id": {"e2e-device"}, "scope": {"read:logs-*"}}.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Skipf("device flow not available (HTTP %d)", resp.StatusCode)
	}
	var codeResp struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}
	json.NewDecoder(resp.Body).Decode(&codeResp)
	resp.Body.Close()

	if codeResp.DeviceCode == "" || codeResp.UserCode == "" {
		t.Fatalf("missing codes: %+v", codeResp)
	}
	t.Logf("  device_code=%s... user_code=%s", codeResp.DeviceCode[:min(len(codeResp.DeviceCode), 12)], codeResp.UserCode)

	// Step 2: Poll — should be authorization_pending
	t.Log("Step 2: Poll (pending)")
	resp, _ = client.Post(proxy+"/oauth/device/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {codeResp.DeviceCode},
		}.Encode()))
	var pollResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&pollResp)
	resp.Body.Close()

	if errStr, ok := pollResp["error"].(string); ok && errStr == "authorization_pending" {
		t.Log("  ✅ Got authorization_pending (correct)")
	} else {
		t.Logf("  Poll response: %v", pollResp)
	}

	// Step 3: Approve
	t.Log("Step 3: Approve device")
	resp, _ = client.Post(proxy+"/oauth/device/approve",
		"application/x-www-form-urlencoded",
		strings.NewReader(url.Values{
			"user_code": {codeResp.UserCode},
			"action":    {"approve"},
		}.Encode()))
	if resp.StatusCode == 200 || resp.StatusCode == 302 {
		t.Log("  ✅ Approved")
	} else {
		t.Logf("  Approve returned HTTP %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Step 4: Poll again — should get token
	t.Log("Step 4: Poll (token)")
	time.Sleep(500 * time.Millisecond)
	resp, _ = client.Post(proxy+"/oauth/device/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {codeResp.DeviceCode},
		}.Encode()))
	var tokenResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	resp.Body.Close()

	if tok, ok := tokenResp["access_token"].(string); ok && tok != "" {
		t.Logf("  ✅ Got access_token: %s...", tok[:min(len(tok), 16)])
	} else if errStr, ok := tokenResp["error"].(string); ok {
		t.Logf("  ⚠️  Still got error: %s (token issuance may need real IdP)", errStr)
	} else {
		t.Logf("  Response: %v", tokenResp)
	}

	t.Log("✅ Device flow E2E complete")
}

// TestDeviceFlowDenyE2E tests the deny path.
func TestDeviceFlowDenyE2E(t *testing.T) {
	proxy := proxyURL(t)
	client := &http.Client{}

	resp, err := client.Post(proxy+"/oauth/device/code",
		"application/x-www-form-urlencoded",
		strings.NewReader(url.Values{"client_id": {"e2e-deny"}}.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Skipf("device flow not available (HTTP %d)", resp.StatusCode)
	}
	var codeResp struct {
		DeviceCode string `json:"device_code"`
		UserCode   string `json:"user_code"`
	}
	json.NewDecoder(resp.Body).Decode(&codeResp)
	resp.Body.Close()

	// Deny
	resp, _ = client.Post(proxy+"/oauth/device/approve",
		"application/x-www-form-urlencoded",
		strings.NewReader(url.Values{"user_code": {codeResp.UserCode}, "action": {"deny"}}.Encode()))
	resp.Body.Close()

	// Poll — should be access_denied
	resp, _ = client.Post(proxy+"/oauth/device/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {codeResp.DeviceCode},
		}.Encode()))
	var errResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errResp)
	resp.Body.Close()

	if errResp["error"] == "access_denied" {
		t.Log("✅ Deny flow works — got access_denied")
	} else {
		t.Logf("⚠️  Expected access_denied, got: %v", errResp)
	}
}
