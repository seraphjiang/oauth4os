package pkce

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestEdge_ConcurrentAuthorize(t *testing.T) {
	h := NewHandler(
		func(clientID string, scopes []string) (string, string) { return "tok", "ref" },
		func(clientID, uri string) bool { return true },
	)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("GET", "/oauth/authorize?client_id=app&code_challenge=abc&code_challenge_method=S256&redirect_uri=http://localhost/cb&scope=read", nil)
			w := httptest.NewRecorder()
			h.Authorize(w, r)
		}()
	}
	wg.Wait()
}

func TestEdge_ExchangeReplayCode(t *testing.T) {
	h := NewHandler(
		func(clientID string, scopes []string) (string, string) { return "tok", "ref" },
		func(clientID, uri string) bool { return true },
	)
	// Try exchanging a code that was never issued
	body := "grant_type=authorization_code&code=never-issued&redirect_uri=http://localhost/cb&client_id=app&code_verifier=test"
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Exchange(w, r)
	if w.Code == 200 {
		t.Error("replayed/fake code should not succeed")
	}
}

func TestEdge_CleanupConcurrent(t *testing.T) {
	h := NewHandler(nil, nil)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.Cleanup()
		}()
	}
	wg.Wait()
}
