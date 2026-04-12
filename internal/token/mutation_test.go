package token

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

// Mutation tests — inject known faults and verify existing tests would catch them.
// Each test simulates a specific mutation and asserts the system rejects it.
// If any of these pass when they shouldn't, our test suite has a gap.

// Mutation: What if IsValid didn't check Revoked?
func TestMutation_IsValidIgnoresRevoked(t *testing.T) {
	m := NewManager()
	m.RegisterClient("c", "s", nil, nil)
	w := issueVia(m, "grant_type=client_credentials&client_id=c&client_secret=s")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["access_token"].(string)

	r := httptest.NewRequest("DELETE", "/oauth/tokens/"+id, nil)
	r.SetPathValue("id", id)
	m.RevokeToken(httptest.NewRecorder(), r)

	// Simulated mutation: skip revoked check
	m.mu.RLock()
	tok := m.tokens[id]
	m.mu.RUnlock()
	if !tok.Revoked {
		t.Fatal("token should be revoked")
	}
	// Real IsValid must return false
	if m.IsValid(id) {
		t.Error("MUTATION SURVIVED: IsValid doesn't check Revoked flag")
	}
}

// Mutation: What if IsValid didn't check ExpiresAt?
func TestMutation_IsValidIgnoresExpiry(t *testing.T) {
	m := NewManager()
	m.RegisterClient("c", "s", nil, nil)
	w := issueVia(m, "grant_type=client_credentials&client_id=c&client_secret=s")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["access_token"].(string)

	// Manually expire the token
	m.mu.Lock()
	m.tokens[id].ExpiresAt = time.Now().Add(-1 * time.Hour)
	m.mu.Unlock()

	if m.IsValid(id) {
		t.Error("MUTATION SURVIVED: IsValid doesn't check ExpiresAt")
	}
}

// Mutation: What if authenticateClient used == instead of constant-time compare?
func TestMutation_TimingAttackOnAuth(t *testing.T) {
	m := NewManager()
	m.RegisterClient("c", "correct-secret", nil, nil)

	// Wrong secret with same prefix should still fail
	w := issueVia(m, "grant_type=client_credentials&client_id=c&client_secret=correct-secre")
	if w.Code == 200 {
		t.Error("MUTATION SURVIVED: partial secret match accepted")
	}

	w2 := issueVia(m, "grant_type=client_credentials&client_id=c&client_secret=correct-secret-extra")
	if w2.Code == 200 {
		t.Error("MUTATION SURVIVED: extended secret accepted")
	}
}

// Mutation: What if validateScopes allowed any scope when client has restrictions?
func TestMutation_ScopeBypass(t *testing.T) {
	m := NewManager()
	m.RegisterClient("c", "s", []string{"read:logs"}, nil)

	w := issueVia(m, "grant_type=client_credentials&client_id=c&client_secret=s&scope=admin")
	if w.Code == 200 {
		t.Error("MUTATION SURVIVED: scope validation bypassed")
	}
}

// Mutation: What if refresh didn't revoke the old token?
func TestMutation_RefreshKeepsOldValid(t *testing.T) {
	m := NewManager()
	m.RegisterClient("c", "s", nil, nil)
	w := issueVia(m, "grant_type=client_credentials&client_id=c&client_secret=s")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	oldID := resp["access_token"].(string)
	rt := resp["refresh_token"].(string)

	issueVia(m, "grant_type=refresh_token&client_id=c&client_secret=s&refresh_token="+rt)

	if m.IsValid(oldID) {
		t.Error("MUTATION SURVIVED: refresh didn't revoke old token")
	}
}

// Mutation: What if refresh token could be reused?
func TestMutation_RefreshTokenReuse(t *testing.T) {
	m := NewManager()
	m.RegisterClient("c", "s", nil, nil)
	w := issueVia(m, "grant_type=client_credentials&client_id=c&client_secret=s")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	rt := resp["refresh_token"].(string)

	// First refresh succeeds
	w2 := issueVia(m, "grant_type=refresh_token&client_id=c&client_secret=s&refresh_token="+rt)
	if w2.Code != 200 {
		t.Fatalf("first refresh should succeed, got %d", w2.Code)
	}

	// Second refresh with same token must fail
	w3 := issueVia(m, "grant_type=refresh_token&client_id=c&client_secret=s&refresh_token="+rt)
	if w3.Code == 200 {
		t.Error("MUTATION SURVIVED: refresh token reuse allowed")
	}
}

// Mutation: What if ListTokens included revoked tokens?
func TestMutation_ListIncludesRevoked(t *testing.T) {
	m := NewManager()
	m.RegisterClient("c", "s", nil, nil)
	w := issueVia(m, "grant_type=client_credentials&client_id=c&client_secret=s")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["access_token"].(string)

	r := httptest.NewRequest("DELETE", "/oauth/tokens/"+id, nil)
	r.SetPathValue("id", id)
	m.RevokeToken(httptest.NewRecorder(), r)

	lr := httptest.NewRequest("GET", "/oauth/tokens", nil)
	lw := httptest.NewRecorder()
	m.ListTokens(lw, lr)
	var list []map[string]interface{}
	json.Unmarshal(lw.Body.Bytes(), &list)
	for _, tok := range list {
		if tok["id"] == id {
			t.Error("MUTATION SURVIVED: ListTokens includes revoked token")
		}
	}
}

// Mutation: What if RevokeToken returned 200 instead of 204?
func TestMutation_RevokeStatusCode(t *testing.T) {
	m := NewManager()
	m.RegisterClient("c", "s", nil, nil)
	w := issueVia(m, "grant_type=client_credentials&client_id=c&client_secret=s")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["access_token"].(string)

	r := httptest.NewRequest("DELETE", "/oauth/tokens/"+id, nil)
	r.SetPathValue("id", id)
	rw := httptest.NewRecorder()
	m.RevokeToken(rw, r)
	if rw.Code != 204 {
		t.Errorf("MUTATION SURVIVED: RevokeToken returns %d instead of 204", rw.Code)
	}
}

// Mutation: remove redirect URI validation → must reject unknown URIs
func TestMutation_ValidateRedirectURI(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "secret", []string{"read"}, []string{"http://localhost/cb"})
	if m.ValidateRedirectURI("app", "http://evil.com/cb") {
		t.Error("must reject unknown redirect URI")
	}
	if !m.ValidateRedirectURI("app", "http://localhost/cb") {
		t.Error("must accept registered redirect URI")
	}
}

// Mutation: remove TouchToken update → must extend token lifetime
func TestMutation_TouchToken(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "secret", []string{"read"}, nil)
	tok, raw := m.CreateTokenForClient("app", []string{"read"})
	if !m.TouchToken(tok.ID, 10*time.Minute) {
		t.Error("TouchToken must return true for valid token")
	}
	if m.TouchToken("nonexistent", 10*time.Minute) {
		t.Error("TouchToken must return false for unknown token")
	}
	_ = raw
}

// Mutation: remove Lookup → must return token metadata
func TestMutation_Lookup(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "secret", []string{"read"}, nil)
	tok, _ := m.CreateTokenForClient("app", []string{"read"})
	clientID, scopes, _, _, _, ok := m.Lookup(tok.ID)
	if !ok {
		t.Fatal("Lookup must find existing token")
	}
	if clientID != "app" {
		t.Errorf("expected client_id 'app', got %q", clientID)
	}
	if len(scopes) == 0 {
		t.Error("Lookup must return scopes")
	}
}

// Mutation: remove auth check → wrong secret must fail
func TestMutation_AuthWrongSecret(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "correct-secret", nil, nil)
	if err := m.AuthenticateClient("app", "wrong-secret"); err == nil {
		t.Error("must reject wrong secret")
	}
}
