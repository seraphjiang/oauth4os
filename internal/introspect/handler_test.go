package introspect

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type mockLookup struct {
	fn func(string) *Response
}

func (m *mockLookup) Introspect(token string) *Response { return m.fn(token) }

func TestIntrospect_ActiveToken(t *testing.T) {
	h := NewHandler(&mockLookup{fn: func(tok string) *Response {
		return &Response{Active: true, ClientID: "app", Scope: "read:logs", TokenType: "Bearer"}
	}})
	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader("token=valid"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Active {
		t.Error("expected active=true")
	}
	if resp.ClientID != "app" {
		t.Errorf("expected app, got %s", resp.ClientID)
	}
}

func TestIntrospect_InactiveToken(t *testing.T) {
	h := NewHandler(&mockLookup{fn: func(tok string) *Response {
		return &Response{Active: false}
	}})
	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader("token=revoked"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Active {
		t.Error("expected active=false")
	}
}

func TestIntrospect_EmptyToken(t *testing.T) {
	h := NewHandler(&mockLookup{fn: func(tok string) *Response { return nil }})
	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader("token="))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Active {
		t.Error("empty token should be inactive")
	}
}

func TestIntrospect_MethodNotAllowed(t *testing.T) {
	h := NewHandler(&mockLookup{fn: func(tok string) *Response { return nil }})
	r := httptest.NewRequest("GET", "/oauth/introspect", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestIntrospect_NilLookupResult(t *testing.T) {
	h := NewHandler(&mockLookup{fn: func(tok string) *Response { return nil }})
	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader("token=unknown"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Active {
		t.Error("nil lookup should return inactive")
	}
}

func TestManagerAdapter_Active(t *testing.T) {
	a := &ManagerAdapter{
		GetToken: func(id string) (string, []string, time.Time, time.Time, bool, bool) {
			return "app", []string{"read:logs"}, time.Now().Add(-1 * time.Hour), time.Now().Add(1 * time.Hour), false, true
		},
	}
	resp := a.Introspect("tok_123")
	if !resp.Active {
		t.Error("expected active")
	}
	if resp.ClientID != "app" {
		t.Errorf("expected app, got %s", resp.ClientID)
	}
}

func TestManagerAdapter_Revoked(t *testing.T) {
	a := &ManagerAdapter{
		GetToken: func(id string) (string, []string, time.Time, time.Time, bool, bool) {
			return "app", nil, time.Now(), time.Now().Add(1 * time.Hour), true, true
		},
	}
	resp := a.Introspect("tok_123")
	if resp.Active {
		t.Error("revoked token should be inactive")
	}
}

func TestManagerAdapter_Expired(t *testing.T) {
	a := &ManagerAdapter{
		GetToken: func(id string) (string, []string, time.Time, time.Time, bool, bool) {
			return "app", nil, time.Now().Add(-2 * time.Hour), time.Now().Add(-1 * time.Hour), false, true
		},
	}
	resp := a.Introspect("tok_123")
	if resp.Active {
		t.Error("expired token should be inactive")
	}
}

func TestManagerAdapter_NotFound(t *testing.T) {
	a := &ManagerAdapter{
		GetToken: func(id string) (string, []string, time.Time, time.Time, bool, bool) {
			return "", nil, time.Time{}, time.Time{}, false, false
		},
	}
	resp := a.Introspect("tok_nope")
	if resp.Active {
		t.Error("missing token should be inactive")
	}
}
