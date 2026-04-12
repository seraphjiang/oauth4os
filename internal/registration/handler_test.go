package registration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterClient(t *testing.T) {
	var registered []string
	h := NewHandler(func(id, secret string, scopes []string) {
		registered = append(registered, id)
	})

	body, _ := json.Marshal(Request{ClientName: "my-agent", Scope: "read:logs-*"})
	req := httptest.NewRequest("POST", "/oauth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ClientID == "" || resp.ClientSecret == "" {
		t.Fatal("expected client_id and client_secret")
	}
	if resp.ClientName != "my-agent" {
		t.Fatalf("expected client_name=my-agent, got %s", resp.ClientName)
	}
	if len(registered) != 1 {
		t.Fatalf("expected 1 registration callback, got %d", len(registered))
	}
}

func TestRegisterMissingName(t *testing.T) {
	h := NewHandler(func(id, secret string, scopes []string) {})
	body, _ := json.Marshal(Request{})
	req := httptest.NewRequest("POST", "/oauth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetClientHidesSecret(t *testing.T) {
	h := NewHandler(func(id, secret string, scopes []string) {})

	// Register first
	body, _ := json.Marshal(Request{ClientName: "test"})
	req := httptest.NewRequest("POST", "/oauth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Register(w, req)

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)

	// GET should not return secret
	mux := http.NewServeMux()
	mux.HandleFunc("GET /oauth/register/{client_id}", h.Get)
	req = httptest.NewRequest("GET", "/oauth/register/"+resp.ClientID, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var getResp Response
	json.NewDecoder(w.Body).Decode(&getResp)
	if getResp.ClientSecret != "" {
		t.Fatal("GET should not return client_secret")
	}
}
