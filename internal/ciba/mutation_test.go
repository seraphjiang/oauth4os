package ciba

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Mutation: remove expiry check → expired requests must return expired_token
func TestMutation_Expiry(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "t", "r" })
	h.mu.Lock()
	h.requests["exp1"] = &authRequest{ExpiresAt: time.Now().Add(-time.Second)}
	h.mu.Unlock()

	mux := http.NewServeMux()
	h.Register(mux)
	w := post(mux, "/oauth/bc-token", url.Values{"auth_req_id": {"exp1"}})
	if w.Code != 400 {
		t.Errorf("expired should return 400, got %d", w.Code)
	}
}

// Mutation: remove Denied check → denied requests must return access_denied
func TestMutation_DenyFlow(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "t", "r" })
	h.mu.Lock()
	h.requests["deny1"] = &authRequest{Denied: true, ExpiresAt: time.Now().Add(time.Minute)}
	h.mu.Unlock()

	mux := http.NewServeMux()
	h.Register(mux)
	w := post(mux, "/oauth/bc-token", url.Values{"auth_req_id": {"deny1"}})
	if w.Code != 400 {
		t.Errorf("denied should return 400, got %d", w.Code)
	}
}

// Mutation: remove authorization_pending → unapproved must return pending
func TestMutation_Pending(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "t", "r" })
	h.mu.Lock()
	h.requests["pend1"] = &authRequest{ExpiresAt: time.Now().Add(time.Minute)}
	h.mu.Unlock()

	mux := http.NewServeMux()
	h.Register(mux)
	w := post(mux, "/oauth/bc-token", url.Values{"auth_req_id": {"pend1"}})
	if w.Code != 400 {
		t.Errorf("pending should return 400, got %d", w.Code)
	}
}

// Mutation: remove delete after approval → approved request must be consumed
func TestMutation_ConsumeOnApproval(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "tok", "rtk" })
	h.mu.Lock()
	h.requests["cons1"] = &authRequest{Approved: true, ClientID: "app", ExpiresAt: time.Now().Add(time.Minute)}
	h.mu.Unlock()

	mux := http.NewServeMux()
	h.Register(mux)
	w := post(mux, "/oauth/bc-token", url.Values{"auth_req_id": {"cons1"}})
	if w.Code != 200 {
		t.Fatalf("approved should return 200, got %d", w.Code)
	}
	// Second poll must fail
	w2 := post(mux, "/oauth/bc-token", url.Values{"auth_req_id": {"cons1"}})
	if w2.Code == 200 {
		t.Error("second poll must not return token — request consumed")
	}
}

// Mutation: Approve with unknown ID → must return 404
func TestMutation_ApproveNotFound(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "t", "r" })
	mux := http.NewServeMux()
	h.Register(mux)
	w := post(mux, "/oauth/bc-approve", url.Values{"auth_req_id": {"nonexistent"}, "action": {"approve"}})
	if w.Code != 404 {
		t.Errorf("approve unknown ID should return 404, got %d", w.Code)
	}
}

// Mutation: remove login_hint check → initiate must require hint
func TestMutation_LoginHintRequired(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)
	body := "client_id=app&scope=openid"
	r := httptest.NewRequest("POST", "/ciba/authorize", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Error("must reject CIBA request without login_hint")
	}
}
