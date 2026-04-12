package exchange

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func mutPost(h *Handler, vals url.Values) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(vals.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func mutHandler() *Handler {
	v := &StaticSubjectValidator{Claims: &SubjectClaims{Subject: "user", Issuer: "https://idp.example.com"}}
	i := &StaticTokenIssuer{TokenID: "tok_123", ExpiresIn: 3600}
	return NewHandler(v, i, "aud")
}

// Mutation: remove subject_token validation → missing token must fail
func TestMutation_MissingSubjectToken(t *testing.T) {
	w := mutPost(mutHandler(), url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:token-exchange"}})
	if w.Code == 200 {
		t.Error("missing subject_token should be rejected")
	}
}

// Mutation: remove token issuance → valid exchange must return access_token
func TestMutation_ValidExchange(t *testing.T) {
	w := mutPost(mutHandler(), url.Values{
		"grant_type":    {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token": {"valid-jwt"},
	})
	if w.Code != 200 {
		t.Fatalf("valid exchange should return 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "access_token") {
		t.Error("response must contain access_token")
	}
}

// Mutation: remove method check → GET must be rejected
func TestMutation_PostOnly(t *testing.T) {
	v := &StaticSubjectValidator{Claims: &SubjectClaims{Subject: "user", Issuer: "https://idp"}}
	i := &StaticTokenIssuer{TokenID: "tok", ExpiresIn: 3600}
	h := NewHandler(v, i, "aud")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/oauth/token", nil))
	if w.Code == 200 {
		t.Error("GET should be rejected, only POST allowed")
	}
}
