package userinfo

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// M1: Missing Authorization header → 401.
func TestMutation_NoAuthHeader(t *testing.T) {
	h := New(func(token string) (string, []string, bool) { return "x", nil, true })
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("missing WWW-Authenticate header")
	}
}

// M2: Invalid token (lookup returns false) → 401.
func TestMutation_InvalidToken(t *testing.T) {
	h := New(func(token string) (string, []string, bool) { return "", nil, false })
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer bad-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// M3: Valid token → 200 with correct sub and scope.
func TestMutation_ValidToken(t *testing.T) {
	h := New(func(token string) (string, []string, bool) {
		return "app-1", []string{"read", "write"}, true
	})
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer good-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["sub"] != "app-1" {
		t.Fatalf("sub: got %v", resp["sub"])
	}
	if resp["scope"] != "read write" {
		t.Fatalf("scope: got %v", resp["scope"])
	}
}

// M4: Content-Type must be application/json.
func TestMutation_ContentType(t *testing.T) {
	h := New(func(token string) (string, []string, bool) { return "x", nil, true })
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
}

// M5: POST method must also work (OIDC §5.3 allows GET and POST).
func TestMutation_PostMethod(t *testing.T) {
	h := New(func(token string) (string, []string, bool) { return "x", nil, true })
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("POST should work, got %d", w.Code)
	}
}

// M6: Bearer prefix is required — "Token xyz" should fail.
func TestMutation_WrongAuthScheme(t *testing.T) {
	h := New(func(token string) (string, []string, bool) { return "x", nil, true })
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Token xyz")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("non-Bearer should be rejected, got %d", w.Code)
	}
}

// M7: Token passed to lookup must be the actual token value, not the full header.
func TestMutation_TokenValuePassedToLookup(t *testing.T) {
	var received string
	h := New(func(token string) (string, []string, bool) {
		received = token
		return "x", nil, true
	})
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer my-secret-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if received != "my-secret-token" {
		t.Fatalf("lookup received %q, expected 'my-secret-token'", received)
	}
}

// M8: Empty scopes → scope field should be empty string, not nil.
func TestMutation_EmptyScopes(t *testing.T) {
	h := New(func(token string) (string, []string, bool) { return "x", nil, true })
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["scope"]; !ok {
		t.Fatal("scope field must be present even when empty")
	}
}
