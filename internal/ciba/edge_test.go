package ciba

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Edge: Initiate requires POST
func TestEdge_InitiateRequiresPOST(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/oauth/bc-authorize", nil))
	if w.Code == 200 {
		t.Error("GET should not be accepted for CIBA initiate")
	}
}

// Edge: Initiate with valid request returns auth_req_id
func TestEdge_InitiateReturnsAuthReqID(t *testing.T) {
	h := NewHandler(func(clientID string, scopes []string) (string, string) {
		return "tok", "ref"
	})
	mux := http.NewServeMux()
	h.Register(mux)
	body := "scope=openid&login_hint=user@example.com"
	r := httptest.NewRequest("POST", "/oauth/bc-authorize", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "auth_req_id") {
		t.Error("response should contain auth_req_id")
	}
}

// Edge: Poll with unknown auth_req_id fails
func TestEdge_PollUnknownFails(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)
	body := "grant_type=urn:openid:params:grant-type:ciba&auth_req_id=unknown-id"
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Error("unknown auth_req_id should fail")
	}
}
