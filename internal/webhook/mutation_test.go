package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mutation: remove Allowed check → denied must return error
func TestMutation_DeniedReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Response{Allowed: false, Reason: "policy"})
	}))
	defer srv.Close()
	a := NewAuthorizer(Config{URL: srv.URL})
	if err := a.Check(Request{ClientID: "app"}); err == nil {
		t.Error("denied webhook must return error")
	}
}

// Mutation: remove status code check → non-200 must error
func TestMutation_Non200Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	a := NewAuthorizer(Config{URL: srv.URL})
	if err := a.Check(Request{ClientID: "app"}); err == nil {
		t.Error("500 from webhook must return error")
	}
}

// Mutation: remove custom headers → headers must be sent
func TestMutation_CustomHeaders(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Api-Key")
		json.NewEncoder(w).Encode(Response{Allowed: true})
	}))
	defer srv.Close()
	a := NewAuthorizer(Config{URL: srv.URL, Headers: map[string]string{"X-Api-Key": "secret123"}})
	a.Check(Request{ClientID: "app"})
	if gotKey != "secret123" {
		t.Errorf("custom header not sent, got %q", gotKey)
	}
}

// Mutation: remove Content-Type → must send application/json
func TestMutation_ContentType(t *testing.T) {
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		json.NewEncoder(w).Encode(Response{Allowed: true})
	}))
	defer srv.Close()
	a := NewAuthorizer(Config{URL: srv.URL})
	a.Check(Request{ClientID: "app"})
	if gotCT != "application/json" {
		t.Errorf("expected application/json, got %q", gotCT)
	}
}

// Mutation: change default timeout → default must be 2s
func TestMutation_DefaultTimeout(t *testing.T) {
	a := NewAuthorizer(Config{URL: "http://localhost:1"})
	if a.client.Timeout.Seconds() != 2 {
		t.Errorf("default timeout should be 2s, got %v", a.client.Timeout)
	}
}

// Mutation: remove body decode → malformed JSON must error
func TestMutation_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()
	a := NewAuthorizer(Config{URL: srv.URL})
	if err := a.Check(Request{ClientID: "app"}); err == nil {
		t.Error("malformed response must return error")
	}
}
