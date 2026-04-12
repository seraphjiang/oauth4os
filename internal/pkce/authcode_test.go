package pkce

import (
	"net/http/httptest"
	"testing"
)

func TestAuthCodeFlowWithoutPKCE(t *testing.T) {
	h := NewHandler(
		func(clientID string, scopes []string) (string, string) { return "tok_123", "rtk_123" },
		func(clientID, redirectURI string) bool { return true },
	)

	authURL := "/oauth/authorize?client_id=svc-1&redirect_uri=https://app/cb&scope=read:logs-*&state=xyz&response_type=code"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", authURL, nil)
	h.Authorize(w, r)

	if w.Code != 200 {
		t.Fatalf("expected consent page 200 without PKCE, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthCodeWithPKCEStillWorks(t *testing.T) {
	h := NewHandler(
		func(clientID string, scopes []string) (string, string) { return "tok_456", "rtk_456" },
		func(clientID, redirectURI string) bool { return true },
	)

	authURL := "/oauth/authorize?client_id=svc-1&redirect_uri=https://app/cb&scope=read:logs-*&code_challenge=abc&code_challenge_method=S256&response_type=code"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", authURL, nil)
	h.Authorize(w, r)

	if w.Code != 200 {
		t.Fatalf("expected consent page 200 with PKCE, got %d", w.Code)
	}
}
