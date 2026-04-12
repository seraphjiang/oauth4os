package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhookAllow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		if req.ClientID != "agent-1" {
			t.Fatalf("expected agent-1, got %s", req.ClientID)
		}
		json.NewEncoder(w).Encode(Response{Allowed: true})
	}))
	defer srv.Close()

	a := NewAuthorizer(Config{URL: srv.URL})
	err := a.Check(Request{ClientID: "agent-1", Action: "GET", Resource: "/logs/_search"})
	if err != nil {
		t.Fatalf("expected allowed, got: %v", err)
	}
}

func TestWebhookDeny(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Response{Allowed: false, Reason: "compliance block"})
	}))
	defer srv.Close()

	a := NewAuthorizer(Config{URL: srv.URL})
	err := a.Check(Request{ClientID: "agent-1", Action: "DELETE", Resource: "/.kibana"})
	if err == nil {
		t.Fatal("expected denied")
	}
}

func TestWebhookUnreachable(t *testing.T) {
	a := NewAuthorizer(Config{URL: "http://127.0.0.1:1", Timeout: 100})
	err := a.Check(Request{ClientID: "agent-1"})
	if err == nil {
		t.Fatal("expected error for unreachable webhook")
	}
}

func TestWebhookBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := NewAuthorizer(Config{URL: srv.URL})
	err := a.Check(Request{ClientID: "agent-1"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestWebhookCustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "secret123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(Response{Allowed: true})
	}))
	defer srv.Close()

	a := NewAuthorizer(Config{URL: srv.URL, Headers: map[string]string{"X-Api-Key": "secret123"}})
	err := a.Check(Request{ClientID: "agent-1"})
	if err != nil {
		t.Fatalf("expected allowed with correct API key, got: %v", err)
	}
}
