package register

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockStore struct {
	registered map[string][]string
}

func (m *mockStore) RegisterClient(id, secret string, scopes, redirectURIs []string) {
	m.registered[id] = scopes
}

func TestRegisterSuccess(t *testing.T) {
	store := &mockStore{registered: make(map[string][]string)}
	h := NewHandler(store, nil)

	body := `{"client_name":"my-agent","redirect_uris":["http://localhost/cb"],"scope":"read:logs-*"}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ClientID == "" || resp.ClientSecret == "" {
		t.Fatal("expected client_id and client_secret")
	}
	if len(store.registered) != 1 {
		t.Fatal("expected 1 registered client")
	}
}

func TestRegisterScopeBlocked(t *testing.T) {
	store := &mockStore{registered: make(map[string][]string)}
	h := NewHandler(store, []string{"read:logs-*"})

	body := `{"client_name":"evil","scope":"admin"}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for disallowed scope, got %d", w.Code)
	}
}

func TestRegisterBadJSON(t *testing.T) {
	h := NewHandler(&mockStore{registered: make(map[string][]string)}, nil)

	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader("{bad"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
