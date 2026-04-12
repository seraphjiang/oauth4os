package ciba

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

func post(mux *http.ServeMux, path string, vals url.Values) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", path, strings.NewReader(vals.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestCIBAFlow(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := http.NewServeMux()
	h.Register(mux)

	// Initiate
	w := post(mux, "/oauth/bc-authorize", url.Values{"client_id": {"svc-1"}, "login_hint": {"user@example.com"}, "scope": {"read:logs-*"}})
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var init map[string]interface{}
	json.NewDecoder(w.Body).Decode(&init)
	reqID := init["auth_req_id"].(string)

	// Poll — pending
	w2 := post(mux, "/oauth/bc-token", url.Values{"auth_req_id": {reqID}})
	var pending map[string]string
	json.NewDecoder(w2.Body).Decode(&pending)
	if pending["error"] != "authorization_pending" {
		t.Fatalf("expected pending, got %s", pending["error"])
	}

	// Approve
	w3 := post(mux, "/oauth/bc-approve", url.Values{"auth_req_id": {reqID}, "action": {"approve"}})
	if w3.Code != 200 {
		t.Fatalf("expected 200, got %d", w3.Code)
	}

	// Poll — token
	w4 := post(mux, "/oauth/bc-token", url.Values{"auth_req_id": {reqID}})
	var tok map[string]interface{}
	json.NewDecoder(w4.Body).Decode(&tok)
	if tok["access_token"] != "access_svc-1" {
		t.Fatalf("expected access_svc-1, got %v", tok["access_token"])
	}
}

func TestCIBADeny(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := http.NewServeMux()
	h.Register(mux)

	w := post(mux, "/oauth/bc-authorize", url.Values{"client_id": {"svc-1"}, "login_hint": {"user@example.com"}})
	var init map[string]interface{}
	json.NewDecoder(w.Body).Decode(&init)
	reqID := init["auth_req_id"].(string)

	post(mux, "/oauth/bc-approve", url.Values{"auth_req_id": {reqID}, "action": {"deny"}})

	w2 := post(mux, "/oauth/bc-token", url.Values{"auth_req_id": {reqID}})
	var err map[string]string
	json.NewDecoder(w2.Body).Decode(&err)
	if err["error"] != "access_denied" {
		t.Fatalf("expected access_denied, got %s", err["error"])
	}
}

func TestCIBAMissingParams(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := http.NewServeMux()
	h.Register(mux)
	w := post(mux, "/oauth/bc-authorize", url.Values{"client_id": {"svc-1"}})
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
