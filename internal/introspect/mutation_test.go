package introspect

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Mutation tests verify that flipping logic in handler.go causes failures.
// Each test targets a specific branch or condition.

// M1: Mutate method check — if we accept GET, this must fail.
func TestMutation_MethodCheckEnforced(t *testing.T) {
	h := NewHandler(&mockLookup{fn: func(tok string) *Response {
		return &Response{Active: true}
	}})
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatal("GET must be rejected")
	}
	// Verify body is empty (no JSON leaked on wrong method)
	if w.Body.Len() > 0 {
		t.Fatal("body should be empty on 405")
	}
}

// M2: Mutate empty-token branch — must return inactive, not call lookup.
func TestMutation_EmptyTokenShortCircuit(t *testing.T) {
	called := false
	h := NewHandler(&mockLookup{fn: func(tok string) *Response {
		called = true
		return &Response{Active: true}
	}})
	r := httptest.NewRequest("POST", "/", strings.NewReader("token="))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if called {
		t.Fatal("lookup must not be called for empty token")
	}
	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Active {
		t.Fatal("empty token must be inactive")
	}
}

// M3: Mutate nil-response guard — nil from lookup must not panic.
func TestMutation_NilResponseGuard(t *testing.T) {
	h := NewHandler(&mockLookup{fn: func(tok string) *Response { return nil }})
	r := httptest.NewRequest("POST", "/", strings.NewReader("token=abc"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r) // must not panic
	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Active {
		t.Fatal("nil lookup result must be inactive")
	}
}

// M4: Mutate ManagerAdapter revoked check.
func TestMutation_AdapterRevokedMustBeInactive(t *testing.T) {
	a := &ManagerAdapter{
		GetToken: func(id string) (string, []string, time.Time, time.Time, bool, bool) {
			return "c", []string{"s"}, time.Now().Add(-time.Hour), time.Now().Add(time.Hour), true, true
		},
	}
	r := a.Introspect("t")
	if r.Active {
		t.Fatal("revoked token must not be active")
	}
}

// M5: Mutate ManagerAdapter expiry check — expired must be inactive.
func TestMutation_AdapterExpiredMustBeInactive(t *testing.T) {
	a := &ManagerAdapter{
		GetToken: func(id string) (string, []string, time.Time, time.Time, bool, bool) {
			return "c", []string{"s"}, time.Now().Add(-2 * time.Hour), time.Now().Add(-time.Second), false, true
		},
	}
	r := a.Introspect("t")
	if r.Active {
		t.Fatal("expired token must not be active")
	}
}

// M6: Mutate ManagerAdapter not-found check.
func TestMutation_AdapterNotFoundMustBeInactive(t *testing.T) {
	a := &ManagerAdapter{
		GetToken: func(id string) (string, []string, time.Time, time.Time, bool, bool) {
			return "", nil, time.Time{}, time.Time{}, false, false
		},
	}
	r := a.Introspect("t")
	if r.Active {
		t.Fatal("not-found token must not be active")
	}
}

// M7: Mutate ManagerAdapter — active token must have correct fields.
func TestMutation_AdapterActiveFieldsCorrect(t *testing.T) {
	now := time.Now()
	exp := now.Add(time.Hour)
	a := &ManagerAdapter{
		GetToken: func(id string) (string, []string, time.Time, time.Time, bool, bool) {
			return "myapp", []string{"read", "write"}, now, exp, false, true
		},
	}
	r := a.Introspect("t")
	if !r.Active {
		t.Fatal("expected active")
	}
	if r.ClientID != "myapp" {
		t.Fatalf("clientID: got %q", r.ClientID)
	}
	if r.Sub != "myapp" {
		t.Fatalf("sub: got %q", r.Sub)
	}
	if r.Scope != "read write" {
		t.Fatalf("scope: got %q", r.Scope)
	}
	if r.TokenType != "Bearer" {
		t.Fatalf("token_type: got %q", r.TokenType)
	}
	if r.Exp != exp.Unix() {
		t.Fatalf("exp: got %d, want %d", r.Exp, exp.Unix())
	}
	if r.Iat != now.Unix() {
		t.Fatalf("iat: got %d, want %d", r.Iat, now.Unix())
	}
}

// M8: Mutate cache TTL — expired cache entry must re-fetch.
func TestMutation_CacheExpiry(t *testing.T) {
	calls := 0
	inner := &mockLookup{fn: func(tok string) *Response {
		calls++
		return &Response{Active: true, ClientID: "c"}
	}}
	c := NewCachedLookup(inner, 1*time.Millisecond)
	c.Introspect("t")
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
	time.Sleep(5 * time.Millisecond)
	c.Introspect("t")
	if calls != 2 {
		t.Fatalf("expected 2 calls after TTL, got %d", calls)
	}
}

// M9: Mutate cache — within TTL must NOT re-fetch.
func TestMutation_CacheHit(t *testing.T) {
	calls := 0
	inner := &mockLookup{fn: func(tok string) *Response {
		calls++
		return &Response{Active: true}
	}}
	c := NewCachedLookup(inner, 10*time.Second)
	c.Introspect("t")
	c.Introspect("t")
	c.Introspect("t")
	if calls != 1 {
		t.Fatalf("cache should prevent re-fetch, got %d calls", calls)
	}
}

// M10: Mutate default TTL — zero TTL must default to 30s, not panic.
func TestMutation_DefaultTTL(t *testing.T) {
	c := NewCachedLookup(&mockLookup{fn: func(tok string) *Response {
		return &Response{Active: true}
	}}, 0)
	if c.ttl != 30*time.Second {
		t.Fatalf("expected 30s default, got %v", c.ttl)
	}
}

// M11: Content-Type header must be application/json.
func TestMutation_ContentTypeJSON(t *testing.T) {
	h := NewHandler(&mockLookup{fn: func(tok string) *Response {
		return &Response{Active: true}
	}})
	r := httptest.NewRequest("POST", "/", strings.NewReader("token=x"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
}
