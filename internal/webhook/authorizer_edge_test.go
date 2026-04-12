package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	a := NewAuthorizer(Config{URL: srv.URL})
	err := a.Check(Request{ClientID: "c1"})
	if err == nil {
		t.Fatal("expected error for malformed response")
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		json.NewEncoder(w).Encode(Response{Allowed: true})
	}))
	defer srv.Close()

	a := NewAuthorizer(Config{URL: srv.URL, Timeout: 50})
	err := a.Check(Request{ClientID: "c1"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestRequestBody(t *testing.T) {
	var received Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		json.NewEncoder(w).Encode(Response{Allowed: true})
	}))
	defer srv.Close()

	a := NewAuthorizer(Config{URL: srv.URL})
	a.Check(Request{
		ClientID: "svc-1",
		Subject:  "user@example.com",
		Scopes:   []string{"read:logs-*"},
		Action:   "POST",
		Resource: "/logs/_search",
		IP:       "10.0.0.1",
	})

	if received.ClientID != "svc-1" || received.Action != "POST" || received.Resource != "/logs/_search" {
		t.Fatalf("request body mismatch: %+v", received)
	}
	if received.IP != "10.0.0.1" {
		t.Fatalf("expected IP 10.0.0.1, got %s", received.IP)
	}
}

func TestDenyWithoutReason(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Response{Allowed: false})
	}))
	defer srv.Close()

	a := NewAuthorizer(Config{URL: srv.URL})
	err := a.Check(Request{ClientID: "c1"})
	if err == nil {
		t.Fatal("expected denied error")
	}
}
