package events

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWebhookSignature(t *testing.T) {
	var mu sync.Mutex
	var gotSig, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		gotSig = r.Header.Get("X-Webhook-Signature")
		gotBody = string(body)
		mu.Unlock()
		w.WriteHeader(200)
	}))
	defer srv.Close()

	key := []byte("my-webhook-secret")
	n := New([]string{srv.URL})
	n.SetSigningKey(key)
	n.Emit(Event{Type: TokenIssued, ClientID: "svc-1"})
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	sig := gotSig
	body := gotBody
	mu.Unlock()

	if !strings.HasPrefix(sig, "sha256=") {
		t.Fatalf("expected sha256= prefix, got %s", sig)
	}

	// Verify HMAC
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(body))
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if sig != expected {
		t.Fatalf("signature mismatch:\n  got:  %s\n  want: %s", sig, expected)
	}
}

func TestNoSignatureWithoutKey(t *testing.T) {
	var mu sync.Mutex
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotSig = r.Header.Get("X-Webhook-Signature")
		mu.Unlock()
		w.WriteHeader(200)
	}))
	defer srv.Close()

	n := New([]string{srv.URL})
	// No SetSigningKey
	n.Emit(Event{Type: TokenIssued, ClientID: "svc-1"})
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if gotSig != "" {
		t.Fatalf("expected no signature without key, got %s", gotSig)
	}
}
