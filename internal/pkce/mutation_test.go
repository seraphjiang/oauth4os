package pkce

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func pkceGet(h *Handler, path string, vals url.Values) *httptest.ResponseRecorder {
	u := path + "?" + vals.Encode()
	w := httptest.NewRecorder()
	h.Authorize(w, httptest.NewRequest("GET", u, nil))
	return w
}

func pkcePost(h *Handler, path string, vals url.Values) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", path, strings.NewReader(vals.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Exchange(w, r)
	return w
}

// Mutation: remove client_id check → missing client_id must be rejected
func TestMutation_ClientIDRequired(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "t", "r" }, nil)
	w := pkceGet(h, "/oauth/authorize", url.Values{
		"response_type":         {"code"},
		"redirect_uri":          {"http://localhost/cb"},
		"code_challenge":        {"abc"},
		"code_challenge_method": {"S256"},
	})
	if w.Code == 200 || w.Code == 302 {
		// Should not proceed without client_id
		body := w.Body.String()
		if !strings.Contains(body, "error") && w.Code != 400 {
			t.Error("missing client_id should produce an error")
		}
	}
}

// Mutation: remove code_challenge check → missing PKCE must be rejected
func TestMutation_PKCERequired(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "t", "r" }, nil)
	w := pkceGet(h, "/oauth/authorize", url.Values{
		"response_type": {"code"},
		"client_id":     {"app"},
		"redirect_uri":  {"http://localhost/cb"},
	})
	// Without code_challenge, should reject or show error
	if w.Code == 302 {
		loc := w.Header().Get("Location")
		if strings.Contains(loc, "code=") {
			t.Error("should not issue code without PKCE challenge")
		}
	}
}

// Mutation: remove code exchange → exchange without valid code must fail
func TestMutation_InvalidCodeExchange(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "t", "r" }, nil)
	w := pkcePost(h, "/oauth/authorize/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"invalid-code"},
		"code_verifier": {"verifier"},
		"redirect_uri":  {"http://localhost/cb"},
	})
	if w.Code == 200 {
		t.Error("invalid code should not return 200")
	}
}

// Mutation: remove Cleanup → must not panic
func TestMutation_Cleanup(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "t", "r" }, nil)
	h.Cleanup() // must not panic
}
