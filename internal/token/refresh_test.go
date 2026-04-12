package token

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

func issueAndRefresh(t *testing.T, m *Manager, clientID, secret string) (origTok *Token, origRefresh string, newResp map[string]interface{}) {
	t.Helper()
	tok, refresh := m.CreateTokenForClient(clientID, []string{"read:logs-*"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
		"client_id":     {clientID},
		"client_secret": {secret},
	}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	m.IssueToken(w, r)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	return tok, refresh, resp
}

func TestRefreshRotatesToken(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)

	oldTok, _, resp := issueAndRefresh(t, m, "svc-1", "secret")

	if resp["access_token"] == nil {
		t.Fatal("expected new access_token")
	}
	if resp["refresh_token"] == nil {
		t.Fatal("expected new refresh_token")
	}
	// Old token should be revoked
	if m.IsValid(oldTok.ID) {
		t.Fatal("old token should be revoked after refresh")
	}
}

func TestRefreshReuseDetection(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)

	_, origRefresh, resp1 := issueAndRefresh(t, m, "svc-1", "secret")
	newToken := resp1["access_token"].(string)

	// Try to reuse the old refresh token
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {origRefresh},
		"client_id":     {"svc-1"},
		"client_secret": {"secret"},
	}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	m.IssueToken(w, r)

	var resp2 map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp2)
	if resp2["error"] != "invalid_grant" {
		t.Fatalf("reused refresh should fail, got %v", resp2)
	}

	// The new token from first refresh should also be revoked (family revocation)
	if m.IsValid(newToken) {
		t.Fatal("entire token family should be revoked on reuse")
	}
}

func TestRefreshWithBasicAuth(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)
	_, refresh := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
	}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth("svc-1", "secret")
	m.IssueToken(w, r)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["access_token"] == nil {
		t.Fatal("refresh with Basic Auth should work")
	}
}

func TestRefreshConcurrent(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)

	// Create 10 tokens, refresh them all concurrently
	refreshTokens := make([]string, 10)
	for i := range refreshTokens {
		_, refreshTokens[i] = m.CreateTokenForClient("svc-1", []string{"read:logs-*"})
	}

	var wg sync.WaitGroup
	for _, rt := range refreshTokens {
		wg.Add(1)
		go func(refresh string) {
			defer wg.Done()
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(url.Values{
				"grant_type":    {"refresh_token"},
				"refresh_token": {refresh},
				"client_id":     {"svc-1"},
				"client_secret": {"secret"},
			}.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			m.IssueToken(w, r)
		}(rt)
	}
	wg.Wait()
}

func TestRefreshScopeDownscope(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*", "admin"}, nil)
	_, refresh := m.CreateTokenForClient("svc-1", []string{"read:logs-*", "admin"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
		"client_id":     {"svc-1"},
		"client_secret": {"secret"},
		"scope":         {"read:logs-*"},
	}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	m.IssueToken(w, r)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["scope"] != "read:logs-*" {
		t.Fatalf("expected narrowed scope, got %v", resp["scope"])
	}
}

func TestRefreshScopeEscalationBlocked(t *testing.T) {
	m := NewManager()
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*", "admin"}, nil)
	_, refresh := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
		"client_id":     {"svc-1"},
		"client_secret": {"secret"},
		"scope":         {"read:logs-* admin"},
	}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	m.IssueToken(w, r)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "invalid_scope" {
		t.Fatalf("scope escalation should be blocked, got %v", resp)
	}
}

func TestRefreshTokenExpiry(t *testing.T) {
	m := NewManager()
	m.SetRefreshTTL(100*time.Millisecond, 90*24*time.Hour)
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)
	_, refresh := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})

	time.Sleep(150 * time.Millisecond)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
		"client_id":     {"svc-1"},
		"client_secret": {"secret"},
	}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	m.IssueToken(w, r)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "invalid_grant" {
		t.Fatalf("expired refresh should fail, got %v", resp)
	}
}

func TestRefreshFamilyAbsoluteLifetime(t *testing.T) {
	m := NewManager()
	m.SetRefreshTTL(1*time.Hour, 100*time.Millisecond) // short absolute lifetime
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)
	_, refresh := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})

	time.Sleep(150 * time.Millisecond)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
		"client_id":     {"svc-1"},
		"client_secret": {"secret"},
	}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	m.IssueToken(w, r)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "invalid_grant" {
		t.Fatalf("family expired should fail, got %v", resp)
	}
}

func TestRefreshBeforeExpiry(t *testing.T) {
	m := NewManager()
	m.SetRefreshTTL(1*time.Hour, 90*24*time.Hour)
	m.RegisterClient("svc-1", "secret", []string{"read:logs-*"}, nil)
	_, refresh := m.CreateTokenForClient("svc-1", []string{"read:logs-*"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
		"client_id":     {"svc-1"},
		"client_secret": {"secret"},
	}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	m.IssueToken(w, r)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["access_token"] == nil {
		t.Fatalf("refresh before expiry should succeed, got %v", resp)
	}
}
