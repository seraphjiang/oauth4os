package token

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestConcurrentRevocation(t *testing.T) {
	m := NewManager()
	m.RegisterClient("test", "secret", []string{"admin"}, nil)

	// Issue a token
	req := httptest.NewRequest("POST", "/oauth/token",
		strings.NewReader("grant_type=client_credentials&client_id=test&client_secret=secret&scope=admin"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	m.IssueToken(w, req)
	if w.Code != 200 {
		t.Fatalf("issue failed: %d %s", w.Code, w.Body.String())
	}

	// Extract token ID
	body := w.Body.String()
	start := strings.Index(body, `"access_token":"`) + len(`"access_token":"`)
	end := strings.Index(body[start:], `"`)
	tokenID := body[start : start+end]

	// Revoke concurrently from 50 goroutines — must not panic or corrupt state
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("DELETE", "/oauth/token/"+tokenID, nil)
			r.SetPathValue("id", tokenID)
			rw := httptest.NewRecorder()
			m.RevokeToken(rw, r)
			// 204 or 200 are both acceptable
			if rw.Code != 204 && rw.Code != 200 {
				t.Errorf("unexpected status: %d", rw.Code)
			}
		}()
	}
	wg.Wait()

	// Verify token is revoked
	m.mu.RLock()
	tok, ok := m.tokens[tokenID]
	m.mu.RUnlock()
	if ok && !tok.Revoked {
		t.Fatal("token should be revoked after concurrent revocations")
	}
}

func TestConcurrentIssueAndRevoke(t *testing.T) {
	m := NewManager()
	m.RegisterClient("test", "secret", []string{"admin"}, nil)

	var wg sync.WaitGroup
	// Issue and revoke in parallel — must not panic
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/oauth/token",
				strings.NewReader("grant_type=client_credentials&client_id=test&client_secret=secret&scope=admin"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			m.IssueToken(w, req)
		}()
	}
	wg.Wait()

	// List all tokens and revoke them concurrently
	req := httptest.NewRequest("GET", "/oauth/tokens", nil)
	w := httptest.NewRecorder()
	m.ListTokens(w, req)

	m.mu.RLock()
	ids := make([]string, 0, len(m.tokens))
	for id := range m.tokens {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	for _, id := range ids {
		wg.Add(1)
		go func(tokenID string) {
			defer wg.Done()
			r := httptest.NewRequest("DELETE", "/oauth/token/"+tokenID, nil)
			r.SetPathValue("id", tokenID)
			rw := httptest.NewRecorder()
			m.RevokeToken(rw, r)
		}(id)
	}
	wg.Wait()
}
