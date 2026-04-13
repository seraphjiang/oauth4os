package exchange

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubValidator struct{}

func (s stubValidator) ValidateSubject(token string) (*SubjectClaims, error) {
	return nil, errors.New("invalid")
}

type stubIssuer struct{}

func (s stubIssuer) IssueExchangeToken(subject, issuer string, scopes []string) (string, int) {
	return "tok_exchange", 3600
}

// Edge: missing grant_type fails
func TestEdge_MissingGrantType(t *testing.T) {
	h := NewHandler(stubValidator{}, stubIssuer{}, "aud")
	body := "subject_token=tok123&subject_token_type=urn:ietf:params:oauth:token-type:access_token"
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Error("missing grant_type should fail")
	}
}

// Edge: GET method rejected
func TestEdge_GETRejected(t *testing.T) {
	h := NewHandler(stubValidator{}, stubIssuer{}, "aud")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/oauth/token", nil))
	if w.Code == 200 {
		t.Error("GET should be rejected")
	}
}

// Edge: missing subject_token fails
func TestEdge_MissingSubjectToken(t *testing.T) {
	h := NewHandler(stubValidator{}, stubIssuer{}, "aud")
	body := "grant_type=urn:ietf:params:oauth:grant-type:token-exchange"
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Error("missing subject_token should fail")
	}
}
