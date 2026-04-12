package exchange

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestDelegationWithActorToken(t *testing.T) {
	h := NewHandler(
		&safeValidator{},
		&safeIssuer{},
		"https://proxy.example.com",
	)

	w := postForm(h, url.Values{
		"grant_type":         {GrantType},
		"subject_token":      {"user-jwt"},
		"subject_token_type": {AccessTokenType},
		"actor_token":        {"service-jwt"},
		"actor_token_type":   {AccessTokenType},
	})

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	// Should include act claim with actor's subject
	act, ok := resp["act"].(map[string]interface{})
	if !ok {
		t.Fatal("expected act claim in delegation response")
	}
	if act["sub"] != "user1" {
		t.Fatalf("expected actor sub=user1, got %v", act["sub"])
	}
}

func TestExchangeWithoutActorToken(t *testing.T) {
	h := NewHandler(&safeValidator{}, &safeIssuer{}, "https://proxy.example.com")

	w := postForm(h, url.Values{
		"grant_type":         {GrantType},
		"subject_token":      {"user-jwt"},
		"subject_token_type": {AccessTokenType},
	})

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	// No act claim without actor_token
	if _, ok := resp["act"]; ok {
		t.Fatal("should not include act claim without actor_token")
	}
}
