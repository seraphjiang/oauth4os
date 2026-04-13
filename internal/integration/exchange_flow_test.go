package integration

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/exchange"
)

// TestTokenExchangeDelegationFlow exercises the full delegation lifecycle:
// validate subject → validate actor → issue delegated token with act claim
func TestTokenExchangeDelegationFlow(t *testing.T) {
	validator := &exchange.StaticSubjectValidator{
		Claims: &exchange.SubjectClaims{
			Subject: "user1",
			Issuer:  "https://idp.example.com",
			Scopes:  []string{"read:logs-*"},
		},
	}
	issuer := &exchange.StaticTokenIssuer{TokenID: "delegated-tok", ExpiresIn: 3600}
	h := exchange.NewHandler(validator, issuer, "https://proxy")

	// Exchange with actor_token (delegation)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/oauth/token",
		strings.NewReader(url.Values{
			"grant_type":         {exchange.GrantType},
			"subject_token":      {"user-jwt"},
			"subject_token_type": {exchange.AccessTokenType},
			"actor_token":        {"service-jwt"},
			"actor_token_type":   {exchange.AccessTokenType},
		}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["access_token"] != "delegated-tok" {
		t.Fatalf("expected delegated-tok, got %v", resp["access_token"])
	}

	// Verify act claim present
	act, ok := resp["act"].(map[string]interface{})
	if !ok {
		t.Fatal("expected act claim in delegation response")
	}
	if act["sub"] != "user1" {
		t.Fatalf("expected act.sub=user1, got %v", act["sub"])
	}
}

// TestTokenExchangeWithoutDelegation verifies no act claim without actor_token
func TestTokenExchangeWithoutDelegation(t *testing.T) {
	validator := &exchange.StaticSubjectValidator{
		Claims: &exchange.SubjectClaims{Subject: "user1", Issuer: "https://idp.example.com"},
	}
	issuer := &exchange.StaticTokenIssuer{TokenID: "plain-tok", ExpiresIn: 3600}
	h := exchange.NewHandler(validator, issuer, "https://proxy")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/oauth/token",
		strings.NewReader(url.Values{
			"grant_type":         {exchange.GrantType},
			"subject_token":      {"user-jwt"},
			"subject_token_type": {exchange.AccessTokenType},
		}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if _, ok := resp["act"]; ok {
		t.Fatal("should not include act claim without actor_token")
	}
}
