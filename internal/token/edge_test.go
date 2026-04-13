package token

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Edge: IssueToken requires POST
func TestEdge_IssueTokenRequiresPOST(t *testing.T) {
	m := NewManager()
	w := httptest.NewRecorder()
	m.IssueToken(w, httptest.NewRequest("GET", "/oauth/token", nil))
	if w.Code == 200 {
		t.Error("GET should not issue token")
	}
}

// Edge: IssueToken with missing grant_type fails
func TestEdge_IssueTokenMissingGrantType(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "secret", []string{"read"}, nil)
	body := "client_id=app&client_secret=secret"
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	m.IssueToken(w, r)
	if w.Code == 200 {
		t.Error("missing grant_type should fail")
	}
}

// Edge: IssueToken with wrong secret fails
func TestEdge_IssueTokenWrongSecret(t *testing.T) {
	m := NewManager()
	m.RegisterClient("app", "correct-secret", []string{"read"}, nil)
	body := "grant_type=client_credentials&client_id=app&client_secret=wrong-secret"
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	m.IssueToken(w, r)
	if w.Code == 200 {
		t.Error("wrong secret should fail")
	}
}

// Edge: ListTokens returns JSON
func TestEdge_ListTokensJSON(t *testing.T) {
	m := NewManager()
	w := httptest.NewRecorder()
	m.ListTokens(w, httptest.NewRequest("GET", "/admin/tokens", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
