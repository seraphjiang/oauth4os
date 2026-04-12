package introspect

import (
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func stubAuth(id, secret string) error {
	if id == "valid" && secret == "secret" {
		return nil
	}
	return errors.New("invalid")
}

type staticLookup struct{ resp *Response }

func (s *staticLookup) Introspect(token string) *Response { return s.resp }

func TestAuthRequired_BasicAuth(t *testing.T) {
	h := NewHandler(&staticLookup{resp: &Response{Active: true, ClientID: "svc-1"}})
	h.SetClientAuth(stubAuth)

	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(url.Values{"token": {"tok-1"}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth("valid", "secret")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("valid basic auth should succeed, got %d", w.Code)
	}
}

func TestAuthRequired_FormAuth(t *testing.T) {
	h := NewHandler(&staticLookup{resp: &Response{Active: true}})
	h.SetClientAuth(stubAuth)

	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(url.Values{
		"token":         {"tok-1"},
		"client_id":     {"valid"},
		"client_secret": {"secret"},
	}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("valid form auth should succeed, got %d", w.Code)
	}
}

func TestAuthRequired_Rejected(t *testing.T) {
	h := NewHandler(&staticLookup{resp: &Response{Active: true}})
	h.SetClientAuth(stubAuth)

	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(url.Values{"token": {"tok-1"}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth("wrong", "creds")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 401 {
		t.Fatalf("invalid auth should return 401, got %d", w.Code)
	}
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("should include WWW-Authenticate header")
	}
}

func TestAuthRequired_NoCredentials(t *testing.T) {
	h := NewHandler(&staticLookup{resp: &Response{Active: true}})
	h.SetClientAuth(stubAuth)

	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(url.Values{"token": {"tok-1"}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 401 {
		t.Fatalf("no credentials should return 401, got %d", w.Code)
	}
}

func TestAuthDisabled_NoCredentialsOK(t *testing.T) {
	h := NewHandler(&staticLookup{resp: &Response{Active: true}})
	// No SetClientAuth — auth disabled

	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(url.Values{"token": {"tok-1"}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("auth disabled should allow unauthenticated, got %d", w.Code)
	}
}
