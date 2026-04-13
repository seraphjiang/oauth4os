package registration

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Edge: Register requires POST
func TestEdge_RegisterRequiresPOST(t *testing.T) {
	h := NewHandler(nil, nil)
	w := httptest.NewRecorder()
	h.Register(w, httptest.NewRequest("GET", "/oauth/register", nil))
	if w.Code == 200 || w.Code == 201 {
		t.Error("GET should not be accepted for registration")
	}
}

// Edge: Register with valid JSON returns client_id
func TestEdge_RegisterReturnsClientID(t *testing.T) {
	registered := false
	h := NewHandler(func(id, secret string, scopes, redirects []string) {
		registered = true
	}, []string{"read", "write"})
	body := `{"client_name":"test-app","redirect_uris":["http://localhost/cb"],"scope":"read"}`
	r := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)
	if w.Code != 201 && w.Code != 200 {
		t.Errorf("expected 201 or 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "client_id") {
		t.Error("response should contain client_id")
	}
	if !registered {
		t.Error("registrar callback should have been called")
	}
}

// Edge: Register with empty body fails
func TestEdge_RegisterEmptyBodyFails(t *testing.T) {
	h := NewHandler(nil, nil)
	r := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)
	if w.Code == 201 {
		t.Error("empty body should not succeed")
	}
}
