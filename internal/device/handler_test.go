package device

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func testIssuer(clientID string, scopes []string) (string, string) {
	return "access_" + clientID, "refresh_" + clientID
}

func formReq(method, path string, vals url.Values) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(vals.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestDeviceFlow(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := http.NewServeMux()
	h.Register(mux)

	// Step 1: Request device code
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, formReq("POST", "/oauth/device/code",
		url.Values{"client_id": {"cli-1"}, "scope": {"read:logs-*"}}))
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	var codeResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&codeResp)
	dc := codeResp["device_code"].(string)
	uc := codeResp["user_code"].(string)
	if dc == "" || uc == "" {
		t.Fatal("missing codes")
	}

	// Step 2: Poll — should be pending
	resp2 := httptest.NewRecorder()
	mux.ServeHTTP(resp2, formReq("POST", "/oauth/device/token",
		url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}, "device_code": {dc}}))
	var pending map[string]string
	json.NewDecoder(resp2.Body).Decode(&pending)
	if pending["error"] != "authorization_pending" {
		t.Fatalf("expected pending, got %s", pending["error"])
	}

	// Step 3: User approves
	resp3 := httptest.NewRecorder()
	mux.ServeHTTP(resp3, formReq("POST", "/oauth/device/approve",
		url.Values{"user_code": {uc}, "action": {"approve"}}))
	if resp3.Code != 200 {
		t.Fatalf("approve: expected 200, got %d", resp3.Code)
	}

	// Step 4: Poll again — should get token
	resp4 := httptest.NewRecorder()
	mux.ServeHTTP(resp4, formReq("POST", "/oauth/device/token",
		url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}, "device_code": {dc}}))
	var tokenResp map[string]interface{}
	json.NewDecoder(resp4.Body).Decode(&tokenResp)
	if tokenResp["access_token"] != "access_cli-1" {
		t.Fatalf("expected access_cli-1, got %v", tokenResp["access_token"])
	}
}

func TestDeviceFlowDeny(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := http.NewServeMux()
	h.Register(mux)

	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, formReq("POST", "/oauth/device/code",
		url.Values{"client_id": {"cli-1"}}))
	var cr map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&cr)

	resp2 := httptest.NewRecorder()
	mux.ServeHTTP(resp2, formReq("POST", "/oauth/device/approve",
		url.Values{"user_code": {cr["user_code"].(string)}, "action": {"deny"}}))

	resp3 := httptest.NewRecorder()
	mux.ServeHTTP(resp3, formReq("POST", "/oauth/device/token",
		url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}, "device_code": {cr["device_code"].(string)}}))
	var errResp map[string]string
	json.NewDecoder(resp3.Body).Decode(&errResp)
	if errResp["error"] != "access_denied" {
		t.Fatalf("expected access_denied, got %s", errResp["error"])
	}
}

func TestMissingClientID(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := http.NewServeMux()
	h.Register(mux)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, formReq("POST", "/oauth/device/code", url.Values{}))
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}
