package exchange

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestHandler(claims *SubjectClaims, valErr error) *Handler {
	return NewHandler(
		&StaticSubjectValidator{Claims: claims, Err: valErr},
		&StaticTokenIssuer{TokenID: "tok_exchanged", ExpiresIn: 3600},
		"https://proxy.example.com",
	)
}

func postForm(h http.Handler, values url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestExchange_Success(t *testing.T) {
	h := newTestHandler(&SubjectClaims{Subject: "user1", Issuer: "https://idp.example.com", Scopes: []string{"read:logs-*"}}, nil)
	rec := postForm(h, url.Values{
		"subject_token":      {"ext-jwt-token"},
		"subject_token_type": {AccessTokenType},
	})
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["access_token"] != "tok_exchanged" {
		t.Fatalf("access_token = %v", resp["access_token"])
	}
	if resp["issued_token_type"] != AccessTokenType {
		t.Fatalf("issued_token_type = %v", resp["issued_token_type"])
	}
	if resp["scope"] != "read:logs-*" {
		t.Fatalf("scope = %v", resp["scope"])
	}
}

func TestExchange_RequestedScopeOverride(t *testing.T) {
	issuer := &StaticTokenIssuer{TokenID: "tok_scoped", ExpiresIn: 3600}
	h := NewHandler(
		&StaticSubjectValidator{Claims: &SubjectClaims{Subject: "user1", Issuer: "idp", Scopes: []string{"admin"}}},
		issuer, "",
	)
	postForm(h, url.Values{
		"subject_token": {"token"},
		"scope":         {"read:logs-*"},
	})
	if len(issuer.LastScopes) != 1 || issuer.LastScopes[0] != "read:logs-*" {
		t.Fatalf("should use requested scope, got %v", issuer.LastScopes)
	}
}

func TestExchange_MissingSubjectToken(t *testing.T) {
	h := newTestHandler(nil, nil)
	rec := postForm(h, url.Values{})
	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestExchange_UnsupportedTokenType(t *testing.T) {
	h := newTestHandler(nil, nil)
	rec := postForm(h, url.Values{
		"subject_token":      {"token"},
		"subject_token_type": {"urn:bad:type"},
	})
	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestExchange_ValidationFailure(t *testing.T) {
	h := newTestHandler(nil, errors.New("expired"))
	rec := postForm(h, url.Values{
		"subject_token": {"bad-token"},
	})
	if rec.Code != 401 {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "invalid_grant" {
		t.Fatalf("error = %s", resp["error"])
	}
}

func TestExchange_DefaultTokenType(t *testing.T) {
	h := newTestHandler(&SubjectClaims{Subject: "u", Issuer: "i"}, nil)
	// No subject_token_type → defaults to access_token
	rec := postForm(h, url.Values{
		"subject_token": {"token"},
	})
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestExchange_IDTokenType(t *testing.T) {
	h := newTestHandler(&SubjectClaims{Subject: "u", Issuer: "i"}, nil)
	rec := postForm(h, url.Values{
		"subject_token":      {"token"},
		"subject_token_type": {IDTokenType},
	})
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestManagerAdapter(t *testing.T) {
	called := false
	adapter := &ManagerAdapter{
		CreateToken: func(clientID string, scopes []string) (string, string) {
			called = true
			if clientID != "user1@https://idp.example.com" {
				t.Fatalf("clientID = %s", clientID)
			}
			return "tok_123", "rtk_123"
		},
	}
	tokenID, expiresIn := adapter.IssueExchangeToken("user1", "https://idp.example.com", []string{"read:logs-*"})
	if !called {
		t.Fatal("CreateToken not called")
	}
	if tokenID != "tok_123" {
		t.Fatalf("tokenID = %s", tokenID)
	}
	if expiresIn != 3600 {
		t.Fatalf("expiresIn = %d", expiresIn)
	}
}
