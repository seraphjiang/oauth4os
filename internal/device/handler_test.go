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

func post(mux *http.ServeMux, path string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestDeviceFlow(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := http.NewServeMux()
	h.Register(mux)

	// Step 1: Request device code
	resp := post(mux, "/oauth/device/code", url.Values{"client_id": {"cli-1"}, "scope": {"read:logs-*"}}.Encode())
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
	resp2 := post(mux, "/oauth/device/token", url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}, "device_code": {dc}}.Encode())
	var pending map[string]string
	json.NewDecoder(resp2.Body).Decode(&pending)
	if pending["error"] != "authorization_pending" {
		t.Fatalf("expected pending, got %s", pending["error"])
	}

	// Step 3: User approves
	resp3 := post(mux, "/oauth/device/approve", url.Values{"user_code": {uc}, "action": {"approve"}}.Encode())
	if resp3.Code != 200 {
		t.Fatalf("expected 200, got %d", resp3.Code)
	}

	// Step 4: Poll again — should get token
	resp4 := post(mux, "/oauth/device/token", url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}, "device_code": {dc}}.Encode())
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

	// Request code
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, httptest.NewRequest("POST", "/oauth/device/code",
		strings.NewReader(url.Values{"client_id": {"cli-1"}}.Encode())))
	var cr map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&cr)

	// Deny
	resp2 := httptest.NewRecorder()
	mux.ServeHTTP(resp2, httptest.NewRequest("POST", "/oauth/device/approve",
		strings.NewReader(url.Values{"user_code": {cr["user_code"].(string)}, "action": {"deny"}}.Encode())))

	// Poll — should be denied
	resp3 := httptest.NewRecorder()
	mux.ServeHTTP(resp3, httptest.NewRequest("POST", "/oauth/device/token",
		strings.NewReader(url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}, "device_code": {cr["device_code"].(string)}}.Encode())))
	var err map[string]string
	json.NewDecoder(resp3.Body).Decode(&err)
	if err["error"] != "access_denied" {
		t.Fatalf("expected access_denied, got %s", err["error"])
	}
}

func TestMissingClientID(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := http.NewServeMux()
	h.Register(mux)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, httptest.NewRequest("POST", "/oauth/device/code", strings.NewReader("")))
	if resp.Code != 400 {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func init() {
	// Set content type for form posts
	http.DefaultTransport = nil
}
