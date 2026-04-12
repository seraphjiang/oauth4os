package ciba

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func post(mux *http.ServeMux, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

// FuzzInitiate ensures the CIBA initiate endpoint never panics.
func FuzzInitiate(f *testing.F) {
	f.Add("login_hint=user@example.com&scope=openid")
	f.Add("")
	f.Add("login_hint=")
	f.Add("scope=" + strings.Repeat("a", 10000))
	f.Add("login_hint=x&binding_message=approve&scope=read:logs-*")
	f.Fuzz(func(t *testing.T, body string) {
		h := NewHandler(func(clientID string, scopes []string) (string, string) {
			return "tok", "rtk"
		})
		mux := http.NewServeMux()
		h.Register(mux)
		r := httptest.NewRequest("POST", "/oauth/bc-authorize", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
	})
}

// FuzzPoll ensures the CIBA poll endpoint never panics.
func FuzzPoll(f *testing.F) {
	f.Add("grant_type=urn:openid:params:grant-type:ciba&auth_req_id=abc")
	f.Add("")
	f.Add("auth_req_id=nonexistent")
	f.Fuzz(func(t *testing.T, body string) {
		h := NewHandler(func(clientID string, scopes []string) (string, string) {
			return "tok", "rtk"
		})
		mux := http.NewServeMux()
		h.Register(mux)
		r := httptest.NewRequest("POST", "/oauth/bc-token", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
	})
}

// Test missing login_hint
func TestInitiateMissingHint(t *testing.T) {
	h := NewHandler(func(clientID string, scopes []string) (string, string) { return "t", "r" })
	mux := http.NewServeMux()
	h.Register(mux)
	w := post(mux, "/oauth/bc-authorize", url.Values{"scope": {"openid"}}.Encode())
	if w.Code != 400 {
		t.Errorf("expected 400 for missing login_hint, got %d", w.Code)
	}
}

// Test poll with invalid auth_req_id
func TestPollInvalidID(t *testing.T) {
	h := NewHandler(func(clientID string, scopes []string) (string, string) { return "t", "r" })
	mux := http.NewServeMux()
	h.Register(mux)
	w := post(mux, "/oauth/bc-token", url.Values{
		"grant_type":  {"urn:openid:params:grant-type:ciba"},
		"auth_req_id": {"nonexistent"},
	}.Encode())
	if w.Code != 400 {
		t.Errorf("expected 400 for invalid auth_req_id, got %d: %s", w.Code, w.Body.String())
	}
}
