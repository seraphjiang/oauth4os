package userinfo

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func stubLookup(token string) (string, []string, bool) {
	if token == "valid-tok" {
		return "svc-1", []string{"read:logs-*", "admin"}, true
	}
	return "", nil, false
}

func TestUserInfoValid(t *testing.T) {
	h := New(stubLookup)
	r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
	r.Header.Set("Authorization", "Bearer valid-tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["sub"] != "svc-1" {
		t.Fatalf("expected sub=svc-1, got %v", resp["sub"])
	}
	if resp["scope"] != "read:logs-* admin" {
		t.Fatalf("expected scopes, got %v", resp["scope"])
	}
}

func TestUserInfoNoToken(t *testing.T) {
	h := New(stubLookup)
	r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("expected WWW-Authenticate header")
	}
}

func TestUserInfoInvalidToken(t *testing.T) {
	h := New(stubLookup)
	r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
	r.Header.Set("Authorization", "Bearer bad-tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUserInfoPOST(t *testing.T) {
	h := New(stubLookup)
	r := httptest.NewRequest("POST", "/oauth/userinfo", nil)
	r.Header.Set("Authorization", "Bearer valid-tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("OIDC allows POST, expected 200, got %d", w.Code)
	}
}

func TestUserInfoContentType(t *testing.T) {
	h := New(stubLookup)
	r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
	r.Header.Set("Authorization", "Bearer valid-tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
}

func BenchmarkUserInfo(b *testing.B) {
	h := New(stubLookup)
	r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
	r.Header.Set("Authorization", "Bearer valid-tok")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}
}
