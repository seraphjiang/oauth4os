package webhook

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSign_Deterministic(t *testing.T) {
	s := NewSender("secret123")
	sig1 := s.Sign([]byte("hello"))
	sig2 := s.Sign([]byte("hello"))
	if sig1 != sig2 {
		t.Error("signatures should be deterministic")
	}
}

func TestSign_DifferentSecrets(t *testing.T) {
	s1 := NewSender("secret1")
	s2 := NewSender("secret2")
	if s1.Sign([]byte("hello")) == s2.Sign([]byte("hello")) {
		t.Error("different secrets should produce different signatures")
	}
}

func TestVerify(t *testing.T) {
	s := NewSender("mysecret")
	body := []byte(`{"event":"token.created"}`)
	sig := s.Sign(body)
	if !s.Verify(body, sig) {
		t.Error("valid signature should verify")
	}
	if s.Verify(body, "wrong") {
		t.Error("wrong signature should not verify")
	}
	if s.Verify([]byte("tampered"), sig) {
		t.Error("tampered body should not verify")
	}
}

func TestSend_IncludesSignature(t *testing.T) {
	s := NewSender("webhooksecret")
	body := []byte(`{"event":"key.rotated"}`)

	var gotSig string
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-Webhook-Signature")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	resp, err := s.Send(srv.URL, body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if !strings.HasPrefix(gotSig, "sha256=") {
		t.Errorf("expected sha256= prefix, got %q", gotSig)
	}
	if gotBody != string(body) {
		t.Errorf("body mismatch: %q", gotBody)
	}
	// Verify the signature
	sig := strings.TrimPrefix(gotSig, "sha256=")
	if !s.Verify([]byte(gotBody), sig) {
		t.Error("sent signature should verify against body")
	}
}
