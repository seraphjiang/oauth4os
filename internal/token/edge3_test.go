package token

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestEdge_ConcurrentIssueToken(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "secret", []string{"read"}, nil)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := "grant_type=client_credentials&client_id=app&client_secret=secret&scope=read"
			r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			m.IssueToken(w, r)
		}()
	}
	wg.Wait()
}

func TestEdge_ConcurrentRevoke(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "secret", []string{"read"}, nil)
	tok, _ := m.CreateTokenForClient("app", []string{"read"})
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := "token=" + tok.ID + "&client_id=app&client_secret=secret"
			r := httptest.NewRequest("POST", "/oauth/revoke", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			m.RevokeRFC7009(w, r)
		}()
	}
	wg.Wait()
}

func TestEdge_TouchNonexistent(t *testing.T) {
	m := NewManager()
	ok := m.TouchToken("nonexistent", 0)
	if ok {
		t.Error("touching nonexistent token should return false")
	}
}
