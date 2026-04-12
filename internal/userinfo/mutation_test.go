package userinfo

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Mutation: remove claims → must return user claims for valid token
func TestMutation_ValidTokenClaims(t *testing.T) {
	h := New(&stubLookup{})
	r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
	r.Header.Set("Authorization", "Bearer valid")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "user123") {
		t.Error("must return user claims")
	}
}

// Mutation: remove auth check → missing token must be rejected
func TestMutation_MissingTokenRejected(t *testing.T) {
	h := New(&stubLookup{})
	r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Error("missing token must be rejected")
	}
}
