package introspect

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type edgeStubLookup struct{}

func (s edgeStubLookup) Introspect(token string) *Response {
	if token == "valid-tok" {
		return &Response{Active: true, ClientID: "app", Scope: "read", Exp: time.Now().Add(time.Hour).Unix()}
	}
	return &Response{Active: false}
}

// Edge: valid token returns active=true
func TestEdge_ValidTokenActive(t *testing.T) {
	h := NewHandler(&edgeStubLookup{})
	body := "token=valid-tok"
	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"active":true`) {
		t.Error("valid token should return active:true")
	}
}

// Edge: invalid token returns active=false
func TestEdge_InvalidTokenInactive(t *testing.T) {
	h := NewHandler(&edgeStubLookup{})
	body := "token=bad-tok"
	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), `"active":false`) {
		t.Error("invalid token should return active:false")
	}
}

// Edge: missing token parameter fails
func TestEdge_MissingToken(t *testing.T) {
	h := NewHandler(&edgeStubLookup{})
	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code == 200 && strings.Contains(w.Body.String(), `"active":true`) {
		t.Error("missing token should not return active:true")
	}
}

// Edge: GET method rejected
func TestEdge_GETRejected(t *testing.T) {
	h := NewHandler(&edgeStubLookup{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/oauth/introspect", nil))
	if w.Code == 200 && strings.Contains(w.Body.String(), `"active":true`) {
		t.Error("GET should not return active token")
	}
}
