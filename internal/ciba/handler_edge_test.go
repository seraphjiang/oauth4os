package ciba

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"
)

func TestCIBAConcurrent(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := newMux(h)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			w := post(mux, "/oauth/bc-authorize", url.Values{
				"client_id":  {"svc-1"},
				"login_hint": {"user@example.com"},
			})
			var init map[string]interface{}
			json.NewDecoder(w.Body).Decode(&init)
			reqID := init["auth_req_id"].(string)

			post(mux, "/oauth/bc-approve", url.Values{"auth_req_id": {reqID}, "action": {"approve"}})

			w2 := post(mux, "/oauth/bc-token", url.Values{"auth_req_id": {reqID}})
			var tok map[string]interface{}
			json.NewDecoder(w2.Body).Decode(&tok)
			if tok["access_token"] == nil {
				t.Errorf("goroutine %d: no token", n)
			}
		}(i)
	}
	wg.Wait()
}

func TestCIBAExpiry(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := newMux(h)

	w := post(mux, "/oauth/bc-authorize", url.Values{"client_id": {"svc-1"}, "login_hint": {"user@example.com"}})
	var init map[string]interface{}
	json.NewDecoder(w.Body).Decode(&init)
	reqID := init["auth_req_id"].(string)

	h.mu.Lock()
	h.requests[reqID].ExpiresAt = time.Now().Add(-1 * time.Second)
	h.mu.Unlock()

	w2 := post(mux, "/oauth/bc-token", url.Values{"auth_req_id": {reqID}})
	var err map[string]string
	json.NewDecoder(w2.Body).Decode(&err)
	if err["error"] != "expired_token" {
		t.Fatalf("expected expired_token, got %s", err["error"])
	}
}

func TestCIBAApproveNotFound(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := newMux(h)
	w := post(mux, "/oauth/bc-approve", url.Values{"auth_req_id": {"nonexistent"}, "action": {"approve"}})
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCIBATokenOneTimeUse(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := newMux(h)

	w := post(mux, "/oauth/bc-authorize", url.Values{"client_id": {"svc-1"}, "login_hint": {"user@example.com"}})
	var init map[string]interface{}
	json.NewDecoder(w.Body).Decode(&init)
	reqID := init["auth_req_id"].(string)

	post(mux, "/oauth/bc-approve", url.Values{"auth_req_id": {reqID}, "action": {"approve"}})

	// First poll gets token
	w2 := post(mux, "/oauth/bc-token", url.Values{"auth_req_id": {reqID}})
	var tok map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&tok)
	if tok["access_token"] == nil {
		t.Fatal("expected token")
	}

	// Second poll fails (one-time)
	w3 := post(mux, "/oauth/bc-token", url.Values{"auth_req_id": {reqID}})
	var err map[string]string
	json.NewDecoder(w3.Body).Decode(&err)
	if err["error"] != "expired_token" {
		t.Fatalf("expected expired_token on reuse, got %s", err["error"])
	}
}

func newMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}
