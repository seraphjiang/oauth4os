package token

import (
	"encoding/json"
	"math/rand"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/quick"
)

// Property-based tests — verify invariants hold for all inputs.

// Property: A revoked token NEVER passes IsValid.
func TestProperty_RevokedTokenNeverValid(t *testing.T) {
	f := func(clientID string, scope string) bool {
		if clientID == "" {
			return true // skip degenerate
		}
		m := NewManager()
		m.RegisterClient(clientID, "s", nil)
		w := issueVia(m, "grant_type=client_credentials&client_id="+clientID+"&client_secret=s&scope="+scope)
		if w.Code != 200 {
			return true
		}
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		id := resp["access_token"].(string)

		// Revoke
		r := httptest.NewRequest("DELETE", "/oauth/tokens/"+id, nil)
		r.SetPathValue("id", id)
		m.RevokeToken(httptest.NewRecorder(), r)

		// INVARIANT: revoked token must never be valid
		return !m.IsValid(id)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 200}); err != nil {
		t.Error(err)
	}
}

// Property: Every issued token appears in ListTokens (until revoked).
func TestProperty_IssuedTokenInList(t *testing.T) {
	f := func(n uint8) bool {
		count := int(n%20) + 1
		m := NewManager()
		m.RegisterClient("c", "s", nil)
		ids := make(map[string]bool)
		for i := 0; i < count; i++ {
			w := issueVia(m, "grant_type=client_credentials&client_id=c&client_secret=s")
			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			ids[resp["access_token"].(string)] = true
		}

		r := httptest.NewRequest("GET", "/oauth/tokens", nil)
		w := httptest.NewRecorder()
		m.ListTokens(w, r)
		var list []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &list)

		listed := make(map[string]bool)
		for _, tok := range list {
			listed[tok["id"].(string)] = true
		}

		// INVARIANT: every issued token must appear in list
		for id := range ids {
			if !listed[id] {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 50}); err != nil {
		t.Error(err)
	}
}

// Property: Refresh token rotation always invalidates the old token.
func TestProperty_RefreshInvalidatesOld(t *testing.T) {
	f := func(seed int64) bool {
		m := NewManager()
		m.RegisterClient("c", "s", nil)
		w := issueVia(m, "grant_type=client_credentials&client_id=c&client_secret=s")
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		oldID := resp["access_token"].(string)
		rt := resp["refresh_token"].(string)

		w2 := issueVia(m, "grant_type=refresh_token&client_id=c&client_secret=s&refresh_token="+rt)
		if w2.Code != 200 {
			return true
		}

		// INVARIANT: old token must be revoked after refresh
		return !m.IsValid(oldID)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

// Property: Invalid client credentials never produce a token.
func TestProperty_InvalidCredsNeverIssue(t *testing.T) {
	f := func(clientID, secret string) bool {
		m := NewManager()
		m.RegisterClient("real", "real-secret", nil)
		w := issueVia(m, "grant_type=client_credentials&client_id="+clientID+"&client_secret="+secret)
		if clientID == "real" && secret == "real-secret" {
			return true // skip the valid case
		}
		// INVARIANT: wrong creds must not return 200
		return w.Code != 200
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 200}); err != nil {
		t.Error(err)
	}
}

// Property: Concurrent issue+revoke never corrupts the token store.
func TestProperty_ConcurrentStoreIntegrity(t *testing.T) {
	m := NewManager()
	m.RegisterClient("c", "s", nil)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var allIDs []string

	// Issue 50 tokens concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := issueVia(m, "grant_type=client_credentials&client_id=c&client_secret=s")
			if w.Code == 200 {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				mu.Lock()
				allIDs = append(allIDs, resp["access_token"].(string))
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// Revoke random half
	rng := rand.New(rand.NewSource(42))
	revoked := make(map[string]bool)
	for _, id := range allIDs {
		if rng.Intn(2) == 0 {
			revoked[id] = true
			r := httptest.NewRequest("DELETE", "/oauth/tokens/"+id, nil)
			r.SetPathValue("id", id)
			m.RevokeToken(httptest.NewRecorder(), r)
		}
	}

	// INVARIANT: revoked tokens invalid, others valid
	for _, id := range allIDs {
		if revoked[id] && m.IsValid(id) {
			t.Errorf("revoked token %s still valid", id[:12])
		}
		if !revoked[id] && !m.IsValid(id) {
			t.Errorf("non-revoked token %s is invalid", id[:12])
		}
	}
}

// Property: Scope validation rejects any scope not in client's allowlist.
func TestProperty_ScopeValidation(t *testing.T) {
	f := func(scope string) bool {
		if strings.ContainsAny(scope, "&= ") {
			return true // skip URL-unsafe
		}
		m := NewManager()
		m.RegisterClient("c", "s", []string{"allowed"})
		w := issueVia(m, "grant_type=client_credentials&client_id=c&client_secret=s&scope="+scope)
		if scope == "allowed" || scope == "" {
			return true // skip valid cases
		}
		// INVARIANT: disallowed scope must be rejected
		return w.Code == 400
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 200}); err != nil {
		t.Error(err)
	}
}
