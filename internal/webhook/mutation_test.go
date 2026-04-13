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

// Mutation: remove HMAC signing → Sign must produce non-empty signature
func TestMutation_SignProducesSignature(t *testing.T) {
	s := NewSender("my-secret")
	sig := s.Sign([]byte(`{"event":"test"}`))
	if sig == "" {
		t.Error("Sign must produce non-empty signature")
	}
}

// Mutation: remove verification → Verify must reject wrong signature
func TestMutation_VerifyRejectsWrong(t *testing.T) {
	s := NewSender("my-secret")
	body := []byte(`{"event":"test"}`)
	if s.Verify(body, "wrong-signature") {
		t.Error("Verify must reject wrong signature")
	}
}

// Mutation: remove round-trip → Sign+Verify must match
func TestMutation_SignVerifyRoundTrip(t *testing.T) {
	s := NewSender("my-secret")
	body := []byte(`{"event":"token.issued"}`)
	sig := s.Sign(body)
	if !s.Verify(body, sig) {
		t.Error("Verify must accept correct signature from Sign")
	}
}

// Mutation: remove secret isolation → different secrets must produce different signatures
func TestMutation_DifferentSecrets(t *testing.T) {
	s1 := NewSender("secret-1")
	s2 := NewSender("secret-2")
	body := []byte(`{"event":"test"}`)
	if s1.Sign(body) == s2.Sign(body) {
		t.Error("different secrets must produce different signatures")
	}
}
