package ciba

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestEdge_ConcurrentInitiate(t *testing.T) {
	h := NewHandler(func(clientID string, scopes []string) (string, string) {
		return "tok", "ref"
	})
	mux := http.NewServeMux()
	h.Register(mux)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := "scope=openid&login_hint=user@example.com&client_id=app"
			r := httptest.NewRequest("POST", "/oauth/bc-authorize", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
		}()
	}
	wg.Wait()
}

func TestEdge_ApproveUnknown(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)
	body := "auth_req_id=unknown-id"
	r := httptest.NewRequest("POST", "/oauth/bc-approve", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Error("approving unknown request should fail")
	}
}
