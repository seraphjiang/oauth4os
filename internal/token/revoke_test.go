package token

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func setupManagerWithClient() *Manager {
	m := NewManager()
	m.RegisterClient("test-client", "test-secret", []string{"read:logs-*"}, nil)
	return m
}

func postForm(handler http.HandlerFunc, path string, vals url.Values) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", path, strings.NewReader(vals.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler(w, r)
	return w
}

func TestRevokeRFC7009_AccessToken(t *testing.T) {
	m := setupManagerWithClient()
	tok, _ := m.CreateTokenForClient("test-client", []string{"read:logs-*"})

	if !m.IsValid(tok.ID) {
		t.Fatal("token should be valid before revocation")
	}

	w := postForm(m.RevokeRFC7009, "/oauth/revoke", url.Values{
		"token":       {tok.ID},
		"client_id":   {"test-client"},
		"client_secret": {"test-secret"},
	})

	// RFC 7009: always 200
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if m.IsValid(tok.ID) {
		t.Fatal("token should be revoked")
	}
}

func TestRevokeRFC7009_RefreshToken(t *testing.T) {
	m := setupManagerWithClient()
	tok, refresh := m.CreateTokenForClient("test-client", []string{"read:logs-*"})

	w := postForm(m.RevokeRFC7009, "/oauth/revoke", url.Values{
		"token":           {refresh},
		"token_type_hint": {"refresh_token"},
		"client_id":       {"test-client"},
		"client_secret":   {"test-secret"},
	})

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// The access token associated with this refresh should be revoked
	if m.IsValid(tok.ID) {
		t.Fatal("token should be revoked via refresh token")
	}
}

func TestRevokeRFC7009_BasicAuth(t *testing.T) {
	m := setupManagerWithClient()
	tok, _ := m.CreateTokenForClient("test-client", []string{"read:logs-*"})

	r := httptest.NewRequest("POST", "/oauth/revoke", strings.NewReader(url.Values{"token": {tok.ID}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth("test-client", "test-secret")
	w := httptest.NewRecorder()
	m.RevokeRFC7009(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if m.IsValid(tok.ID) {
		t.Fatal("token should be revoked via basic auth")
	}
}

func TestRevokeRFC7009_InvalidClient(t *testing.T) {
	m := setupManagerWithClient()

	w := postForm(m.RevokeRFC7009, "/oauth/revoke", url.Values{
		"token":         {"anything"},
		"client_id":     {"test-client"},
		"client_secret": {"wrong-secret"},
	})

	if w.Code != 401 {
		t.Fatalf("expected 401 for bad client auth, got %d", w.Code)
	}
}

func TestRevokeRFC7009_EmptyToken(t *testing.T) {
	m := setupManagerWithClient()

	w := postForm(m.RevokeRFC7009, "/oauth/revoke", url.Values{})

	// RFC 7009: empty token still returns 200
	if w.Code != 200 {
		t.Fatalf("expected 200 for empty token, got %d", w.Code)
	}
}

func TestRevokeRFC7009_UnknownToken(t *testing.T) {
	m := setupManagerWithClient()

	w := postForm(m.RevokeRFC7009, "/oauth/revoke", url.Values{
		"token": {"nonexistent-token"},
	})

	// RFC 7009: always 200, even for unknown tokens (prevents scanning)
	if w.Code != 200 {
		t.Fatalf("expected 200 for unknown token, got %d", w.Code)
	}
}
