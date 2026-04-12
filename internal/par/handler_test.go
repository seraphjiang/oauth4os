package par

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPushAndResolve(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)

	body := url.Values{
		"client_id":             {"app-1"},
		"scope":                 {"read:logs-*"},
		"redirect_uri":         {"https://app.example.com/callback"},
		"state":                 {"xyz"},
		"code_challenge":        {"abc123"},
		"code_challenge_method": {"S256"},
	}
	r := httptest.NewRequest("POST", "/oauth/par", strings.NewReader(body.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	uri := resp["request_uri"].(string)
	if !strings.HasPrefix(uri, "urn:ietf:params:oauth:request_uri:") {
		t.Fatalf("unexpected URI format: %s", uri)
	}

	// Resolve
	clientID, scopes, redirectURI, state, cc, cm, ok := h.Resolve(uri)
	if !ok {
		t.Fatal("expected resolve to succeed")
	}
	if clientID != "app-1" || redirectURI != "https://app.example.com/callback" || state != "xyz" {
		t.Fatalf("unexpected values: %s %s %s", clientID, redirectURI, state)
	}
	if len(scopes) != 1 || scopes[0] != "read:logs-*" {
		t.Fatalf("unexpected scopes: %v", scopes)
	}
	if cc != "abc123" || cm != "S256" {
		t.Fatalf("unexpected PKCE: %s %s", cc, cm)
	}

	// Second resolve should fail (one-time use)
	_, _, _, _, _, _, ok2 := h.Resolve(uri)
	if ok2 {
		t.Fatal("expected one-time use")
	}
}

func TestMissingClientID(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)

	r := httptest.NewRequest("POST", "/oauth/par", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBasicAuth(t *testing.T) {
	authed := false
	h := NewHandler(func(id, secret string) error {
		authed = true
		return nil
	})
	mux := http.NewServeMux()
	h.Register(mux)

	r := httptest.NewRequest("POST", "/oauth/par", strings.NewReader("scope=read:logs-*"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth("app-1", "secret")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !authed {
		t.Fatal("expected client auth to be called")
	}
}
