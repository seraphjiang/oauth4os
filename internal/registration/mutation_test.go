package registration

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func regPost(h *Handler, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)
	return w
}

// Mutation: remove client_name check → missing name must be rejected
func TestMutation_ClientNameRequired(t *testing.T) {
	h := NewHandler(func(id, secret string, scopes, uris []string) {}, nil)
	w := regPost(h, `{"scope":"read:logs-*"}`)
	if w.Code == 201 {
		t.Error("missing client_name should be rejected")
	}
}

// Mutation: remove client_id generation → response must include client_id
func TestMutation_ClientIDGenerated(t *testing.T) {
	h := NewHandler(func(id, secret string, scopes, uris []string) {}, nil)
	w := regPost(h, `{"client_name":"test-app"}`)
	if w.Code != 201 {
		t.Skipf("registration returned %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["client_id"] == nil || resp["client_id"] == "" {
		t.Error("response must include client_id")
	}
}

// Mutation: remove client_secret generation → response must include secret
func TestMutation_SecretGenerated(t *testing.T) {
	h := NewHandler(func(id, secret string, scopes, uris []string) {}, nil)
	w := regPost(h, `{"client_name":"test-app"}`)
	if w.Code != 201 {
		t.Skipf("registration returned %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["client_secret"] == nil || resp["client_secret"] == "" {
		t.Error("response must include client_secret")
	}
}

// Mutation: remove 201 status → successful registration must return 201
func TestMutation_201Status(t *testing.T) {
	h := NewHandler(func(id, secret string, scopes, uris []string) {}, nil)
	w := regPost(h, `{"client_name":"test-app"}`)
	if w.Code != 201 {
		t.Errorf("successful registration must return 201, got %d", w.Code)
	}
}

// Mutation: remove JSON content type → response must be application/json
func TestMutation_JSONContentType(t *testing.T) {
	h := NewHandler(nil, nil)
	mux := http.NewServeMux()
	h.Register(mux)
	body := `{"client_name":"test-app","redirect_uris":["http://localhost/cb"]}`
	r := httptest.NewRequest("POST", "/register", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if !strings.Contains(w.Header().Get("Content-Type"), "json") {
		t.Error("registration response must be JSON")
	}
}
