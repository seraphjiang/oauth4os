package exchange

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func postExchange(h *Handler, vals url.Values) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(vals.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// Mutation: remove grant_type check → wrong grant_type must be rejected
func TestMutation_GrantType(t *testing.T) {
	h := NewHandler(&StaticSubjectValidator{Sub: "user"}, &StaticTokenIssuer{Token: "tok", Expiry: 3600}, "aud")
	w := postExchange(h, url.Values{
		"grant_type":    {"authorization_code"},
		"subject_token": {"valid"},
	})
	if w.Code == 200 {
		t.Error("wrong grant_type should be rejected")
	}
}

// Mutation: remove subject_token check → missing token must be rejected
func TestMutation_SubjectTokenRequired(t *testing.T) {
	h := NewHandler(&StaticSubjectValidator{Sub: "user"}, &StaticTokenIssuer{Token: "tok", Expiry: 3600}, "aud")
	w := postExchange(h, url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:token-exchange"},
	})
	if w.Code == 200 {
		t.Error("missing subject_token should be rejected")
	}
}

// Mutation: remove POST check → GET must not work
func TestMutation_PostOnly(t *testing.T) {
	h := NewHandler(&StaticSubjectValidator{Sub: "user"}, &StaticTokenIssuer{Token: "tok", Expiry: 3600}, "aud")
	r := httptest.NewRequest("GET", "/oauth/token?grant_type=urn:ietf:params:oauth:grant-type:token-exchange&subject_token=x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Error("GET should not be accepted for token exchange")
	}
}

// Mutation: valid exchange must return access_token
func TestMutation_ValidExchange(t *testing.T) {
	h := NewHandler(&StaticSubjectValidator{Sub: "user"}, &StaticTokenIssuer{Token: "tok_123", Expiry: 3600}, "aud")
	w := postExchange(h, url.Values{
		"grant_type":    {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token": {"valid-jwt"},
	})
	if w.Code != 200 {
		t.Fatalf("valid exchange should return 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "access_token") {
		t.Error("response must contain access_token")
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("response must be application/json")
	}
}
