package userinfo

import (
	"net/http/httptest"
	"testing"
)

// Edge: missing Authorization header returns 401
func TestEdge_MissingAuthHeader(t *testing.T) {
	h := New(stubLookup)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/oauth/userinfo", nil))
	if w.Code != 401 {
		t.Errorf("missing auth should return 401, got %d", w.Code)
	}
}

// Edge: invalid bearer token returns 401
func TestEdge_InvalidBearerToken(t *testing.T) {
	h := New(stubLookup)
	r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
	r.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Errorf("invalid token should return 401, got %d", w.Code)
	}
}

// Edge: non-Bearer auth scheme returns 401
func TestEdge_NonBearerScheme(t *testing.T) {
	h := New(stubLookup)
	r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
	r.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Errorf("non-Bearer should return 401, got %d", w.Code)
	}
}
