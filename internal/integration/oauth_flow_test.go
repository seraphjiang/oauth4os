// Package integration provides end-to-end OAuth flow tests.
package integration

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/token"
)

// TestFullOAuthFlow exercises the complete lifecycle:
// register client → issue token → use token → revoke → verify 401
func TestFullOAuthFlow(t *testing.T) {
	mgr := token.NewManager()

	// 1. Register client
	mgr.RegisterClient("test-svc", "s3cret", []string{"read:logs-*", "write:logs"}, nil)

	// 2. Issue token via client_credentials
	issueReq := httptest.NewRequest("POST", "/oauth/token",
		strings.NewReader(url.Values{
			"grant_type":    {"client_credentials"},
			"client_id":     {"test-svc"},
			"client_secret": {"s3cret"},
			"scope":         {"read:logs-*"},
		}.Encode()))
	issueReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mgr.IssueToken(w, issueReq)

	if w.Code != 200 {
		t.Fatalf("issue: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var issueResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&issueResp)
	accessToken, ok := issueResp["access_token"].(string)
	if !ok || accessToken == "" {
		t.Fatal("issue: no access_token in response")
	}

	// 3. Use token — verify it's valid
	if !mgr.IsValid(accessToken) {
		t.Fatal("use: token should be valid after issuance")
	}

	clientID, scopes, _, _, revoked, found := mgr.Lookup(accessToken)
	if !found {
		t.Fatal("use: token not found via Lookup")
	}
	if clientID != "test-svc" {
		t.Fatalf("use: expected client_id=test-svc, got %s", clientID)
	}
	if len(scopes) != 1 || scopes[0] != "read:logs-*" {
		t.Fatalf("use: expected scopes=[read:logs-*], got %v", scopes)
	}
	if revoked {
		t.Fatal("use: token should not be revoked")
	}

	// 4. Revoke token
	revokeReq := httptest.NewRequest("DELETE", "/oauth/token/"+accessToken, nil)
	revokeReq.SetPathValue("id", accessToken)
	rw := httptest.NewRecorder()
	mgr.RevokeToken(rw, revokeReq)

	if rw.Code != 204 && rw.Code != 200 {
		t.Fatalf("revoke: expected 204/200, got %d", rw.Code)
	}

	// 5. Verify 401 — token no longer valid
	if mgr.IsValid(accessToken) {
		t.Fatal("post-revoke: token should be invalid")
	}

	_, _, _, _, revoked, found = mgr.Lookup(accessToken)
	if !found {
		t.Fatal("post-revoke: token should still exist in store")
	}
	if !revoked {
		t.Fatal("post-revoke: token should be marked revoked")
	}
}

// TestFullOAuthFlowWithRefresh exercises: issue → refresh → revoke refresh → verify
func TestFullOAuthFlowWithRefresh(t *testing.T) {
	mgr := token.NewManager()
	mgr.RegisterClient("app", "pw", []string{"read:logs-*"}, nil)

	// Issue
	w := issueToken(t, mgr, "app", "pw", "read:logs-*")
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	accessToken := resp["access_token"].(string)
	refreshToken, _ := resp["refresh_token"].(string)
	if refreshToken == "" {
		t.Fatal("expected refresh_token")
	}

	// Refresh
	refreshReq := httptest.NewRequest("POST", "/oauth/token",
		strings.NewReader(url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {refreshToken},
			"client_id":     {"app"},
			"client_secret": {"pw"},
		}.Encode()))
	refreshReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rw := httptest.NewRecorder()
	mgr.IssueToken(rw, refreshReq)

	if rw.Code != 200 {
		t.Fatalf("refresh: expected 200, got %d: %s", rw.Code, rw.Body.String())
	}

	var refreshResp map[string]interface{}
	json.NewDecoder(rw.Body).Decode(&refreshResp)
	newToken := refreshResp["access_token"].(string)

	// Old token revoked after rotation
	if mgr.IsValid(accessToken) {
		t.Fatal("old token should be invalid after refresh rotation")
	}

	// New token valid
	if !mgr.IsValid(newToken) {
		t.Fatal("new token should be valid")
	}
}

// TestFullOAuthFlowBadCredentials verifies auth rejection
func TestFullOAuthFlowBadCredentials(t *testing.T) {
	mgr := token.NewManager()
	mgr.RegisterClient("svc", "correct", []string{"read:logs-*"}, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/oauth/token",
		strings.NewReader(url.Values{
			"grant_type":    {"client_credentials"},
			"client_id":     {"svc"},
			"client_secret": {"wrong"},
			"scope":         {"read:logs-*"},
		}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mgr.IssueToken(w, req)

	if w.Code == 200 {
		t.Fatal("should reject bad credentials")
	}
}

func issueToken(t *testing.T, mgr *token.Manager, clientID, secret, scope string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/oauth/token",
		strings.NewReader(url.Values{
			"grant_type":    {"client_credentials"},
			"client_id":     {clientID},
			"client_secret": {secret},
			"scope":         {scope},
		}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mgr.IssueToken(w, req)
	if w.Code != 200 {
		t.Fatalf("issueToken: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	return w
}
