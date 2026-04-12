package token

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestConcurrentRevocation hammers revoke + isValid + list from multiple goroutines.
// Run with: go test -race -run TestConcurrent ./internal/token/
func TestConcurrentRevocation(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "secret", []string{"read:logs"}, nil)

	// Issue 100 tokens
	ids := make([]string, 100)
	for i := range ids {
		w := issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=secret&scope=read:logs")
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		ids[i] = resp["access_token"].(string)
	}

	var wg sync.WaitGroup

	// Concurrent revocations
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			r := httptest.NewRequest("DELETE", "/oauth/tokens/"+ids[idx], nil)
			r.SetPathValue("id", ids[idx])
			m.RevokeToken(httptest.NewRecorder(), r)
		}(i)
	}

	// Concurrent IsValid checks
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			m.IsValid(ids[idx])
		}(idx(i, len(ids)))
	}

	// Concurrent ListTokens
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("GET", "/oauth/tokens", nil)
			m.ListTokens(httptest.NewRecorder(), r)
		}()
	}

	// Concurrent Lookup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			m.Lookup(ids[idx])
		}(idx(i, len(ids)))
	}

	wg.Wait()

	// Verify: first 50 revoked, rest valid
	for i := 0; i < 50; i++ {
		if m.IsValid(ids[i]) {
			t.Errorf("token %d should be revoked", i)
		}
	}
	for i := 50; i < 100; i++ {
		if !m.IsValid(ids[i]) {
			t.Errorf("token %d should still be valid", i)
		}
	}
}

// TestConcurrentIssueAndRevoke issues and revokes tokens simultaneously.
func TestConcurrentIssueAndRevoke(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "secret", []string{"read:logs"}, nil)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var issued []string

	// Issue tokens concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=secret&scope=read:logs")
			if w.Code == 200 {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				mu.Lock()
				issued = append(issued, resp["access_token"].(string))
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(issued) != 50 {
		t.Fatalf("expected 50 tokens, got %d", len(issued))
	}

	// Revoke all concurrently
	for _, id := range issued {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			r := httptest.NewRequest("DELETE", "/oauth/tokens/"+id, nil)
			r.SetPathValue("id", id)
			m.RevokeToken(httptest.NewRecorder(), r)
		}(id)
	}
	wg.Wait()

	// All should be revoked
	for _, id := range issued {
		if m.IsValid(id) {
			t.Errorf("token %s should be revoked", id)
		}
	}
}

// TestConcurrentRefreshRevoke races refresh against revocation.
func TestConcurrentRefreshRevoke(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "secret", []string{"read:logs"}, nil)

	w := issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=secret&scope=read:logs")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	tokenID := resp["access_token"].(string)
	refreshTok := resp["refresh_token"].(string)

	var wg sync.WaitGroup

	// Race: revoke the token
	wg.Add(1)
	go func() {
		defer wg.Done()
		r := httptest.NewRequest("DELETE", "/oauth/tokens/"+tokenID, nil)
		r.SetPathValue("id", tokenID)
		m.RevokeToken(httptest.NewRecorder(), r)
	}()

	// Race: refresh the token
	wg.Add(1)
	go func() {
		defer wg.Done()
		issueVia(m, "grant_type=refresh_token&client_id=app&client_secret=secret&refresh_token="+refreshTok)
	}()

	wg.Wait()
	// No panic = pass. The old token should be revoked regardless.
	if m.IsValid(tokenID) {
		t.Error("original token should be revoked")
	}
}

// TestDoubleRevoke ensures revoking the same token twice is safe.
func TestDoubleRevoke(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "secret", []string{"read:logs"}, nil)

	w := issueVia(m, "grant_type=client_credentials&client_id=app&client_secret=secret&scope=read:logs")
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["access_token"].(string)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("DELETE", "/oauth/tokens/"+id, nil)
			r.SetPathValue("id", id)
			rw := httptest.NewRecorder()
			m.RevokeToken(rw, r)
			if rw.Code != 204 {
				t.Errorf("expected 204, got %d", rw.Code)
			}
		}()
	}
	wg.Wait()
}

func idx(i, max int) int { return i % max }

// Helper already defined in manager_test.go but needed if run standalone
func issueViaRace(m *Manager, form string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	m.IssueToken(w, r)
	return w
}
