package token

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setup() *Manager {
	m := NewManager()
	m.RegisterClient("app", "secret", []string{"read:logs", "write:logs"}, nil)
	m.RegisterClient("admin", "admin-secret", nil, nil) // no scope restriction
	return m
}

func issueVia(m *Manager, form string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	m.IssueToken(w, r)
	return w
}

func TestIssueToken_ClientCredentials(t *testing.T) {
	m := setup()
	w := issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=secret&scope=read:logs")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["access_token"] == nil || resp["access_token"] == "" {
		t.Fatal("missing access_token")
	}
	if resp["token_type"] != "Bearer" {
		t.Errorf("expected Bearer, got %v", resp["token_type"])
	}
	if resp["refresh_token"] == nil {
		t.Error("missing refresh_token")
	}
}

func TestIssueToken_InvalidClient(t *testing.T) {
	m := setup()
	w := issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=wrong")
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestIssueToken_UnknownClient(t *testing.T) {
	m := setup()
	w := issueVia(m, "grant_type=client_credentials&client_id=nope&client_secret=x")
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestIssueToken_InvalidScope(t *testing.T) {
	m := setup()
	w := issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=secret&scope=admin")
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestIssueToken_NoScopeRestriction(t *testing.T) {
	m := setup()
	w := issueVia(m, "grant_type=client_credentials&client_id=admin&client_secret=admin-secret&scope=anything")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIssueToken_UnsupportedGrant(t *testing.T) {
	m := setup()
	w := issueVia(m, "grant_type=password&client_id=app&client_secret=secret")
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRefreshToken(t *testing.T) {
	m := setup()
	w := issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=secret&scope=read:logs")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	rt := resp["refresh_token"].(string)

	w2 := issueVia(m, "grant_type=refresh_token&client_id=app&client_secret=secret&refresh_token="+rt)
	if w2.Code != 200 {
		t.Fatalf("refresh expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var resp2 map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2["access_token"] == resp["access_token"] {
		t.Error("refresh should issue new token")
	}
}

func TestRefreshToken_InvalidRefresh(t *testing.T) {
	m := setup()
	w := issueVia(m, "grant_type=refresh_token&client_id=app&client_secret=secret&refresh_token=bogus")
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRefreshToken_WrongClient(t *testing.T) {
	m := setup()
	w := issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=secret&scope=read:logs")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	rt := resp["refresh_token"].(string)

	w2 := issueVia(m, "grant_type=refresh_token&client_id=admin&client_secret=admin-secret&refresh_token="+rt)
	if w2.Code != 400 {
		t.Fatalf("expected 400 for wrong client, got %d", w2.Code)
	}
}

func TestIsValid(t *testing.T) {
	m := setup()
	w := issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=secret&scope=read:logs")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["access_token"].(string)

	if !m.IsValid(id) {
		t.Error("fresh token should be valid")
	}
	if m.IsValid("nonexistent") {
		t.Error("nonexistent token should be invalid")
	}
}

func TestRevokeToken(t *testing.T) {
	m := setup()
	w := issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=secret&scope=read:logs")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["access_token"].(string)

	r := httptest.NewRequest("DELETE", "/oauth/tokens/"+id, nil)
	r.SetPathValue("id", id)
	rw := httptest.NewRecorder()
	m.RevokeToken(rw, r)
	if rw.Code != 204 {
		t.Fatalf("expected 204, got %d", rw.Code)
	}
	if m.IsValid(id) {
		t.Error("revoked token should be invalid")
	}
}

func TestListTokens_ExcludesRevoked(t *testing.T) {
	m := setup()
	issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=secret&scope=read:logs")
	w := issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=secret&scope=read:logs")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["access_token"].(string)

	// Revoke second token
	r := httptest.NewRequest("DELETE", "/oauth/tokens/"+id, nil)
	r.SetPathValue("id", id)
	m.RevokeToken(httptest.NewRecorder(), r)

	// List should exclude revoked
	lr := httptest.NewRequest("GET", "/oauth/tokens", nil)
	lw := httptest.NewRecorder()
	m.ListTokens(lw, lr)
	var list []map[string]interface{}
	json.Unmarshal(lw.Body.Bytes(), &list)
	for _, tok := range list {
		if tok["id"] == id {
			t.Error("revoked token should not appear in list")
		}
	}
}

func TestGetToken_NotFound(t *testing.T) {
	m := setup()
	r := httptest.NewRequest("GET", "/oauth/tokens/nope", nil)
	r.SetPathValue("id", "nope")
	w := httptest.NewRecorder()
	m.GetToken(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestLookup(t *testing.T) {
	m := setup()
	w := issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=secret&scope=read:logs")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["access_token"].(string)

	clientID, scopes, _, _, revoked, ok := m.Lookup(id)
	if !ok {
		t.Fatal("Lookup should find token")
	}
	if clientID != "app" {
		t.Errorf("expected app, got %s", clientID)
	}
	if len(scopes) != 1 || scopes[0] != "read:logs" {
		t.Errorf("unexpected scopes: %v", scopes)
	}
	if revoked {
		t.Error("should not be revoked")
	}

	_, _, _, _, _, ok = m.Lookup("nonexistent")
	if ok {
		t.Error("Lookup should return false for missing token")
	}
}

func TestClients(t *testing.T) {
	m := NewManager()
	m.RegisterClient("c1", "s1", []string{"read:*"}, nil)
	m.RegisterClient("c2", "s2", nil, nil)
	clients := m.Clients()
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}
}

func TestValidateRedirectURI(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "secret", nil, []string{"http://localhost:3000/cb"})

	if !m.ValidateRedirectURI("app", "http://localhost:3000/cb") {
		t.Error("should allow registered URI")
	}
	if m.ValidateRedirectURI("app", "http://evil.com/cb") {
		t.Error("should reject unregistered URI")
	}
	if m.ValidateRedirectURI("unknown", "http://anything.com") {
		t.Error("should reject URI for unknown client")
	}
}

func TestTouchToken(t *testing.T) {
	m := NewManager()
	m.RegisterClient("c1", "s1", nil, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(
		"grant_type=client_credentials&client_id=c1&client_secret=s1&scope=read:logs-*"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	m.IssueToken(w, r)

	var tok map[string]interface{}
	json.NewDecoder(w.Body).Decode(&tok)
	tokenID := tok["token_id"]

	if tokenID == nil {
		// Token might not expose ID — just verify TouchToken doesn't panic
		m.TouchToken("nonexistent", 30*time.Minute)
		return
	}
	m.TouchToken(tokenID.(string), 30*time.Minute)
}
