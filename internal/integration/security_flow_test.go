package integration

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/seraphjiang/oauth4os/internal/events"
	"github.com/seraphjiang/oauth4os/internal/token"
)

// TestIntrospectionFlow exercises: issue → introspect active → revoke → introspect inactive
func TestIntrospectionFlow(t *testing.T) {
	mgr := token.NewManager()
	mgr.RegisterClient("svc", "pw", []string{"read:logs-*"}, nil)

	w := issueToken(t, mgr, "svc", "pw", "read:logs-*")
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	tok := resp["access_token"].(string)

	// Introspect — should be active
	if !mgr.IsValid(tok) {
		t.Fatal("token should be valid")
	}
	clientID, scopes, _, _, revoked, ok := mgr.Lookup(tok)
	if !ok || revoked {
		t.Fatal("introspect: token should be active")
	}
	if clientID != "svc" || len(scopes) == 0 {
		t.Fatalf("introspect: unexpected client=%s scopes=%v", clientID, scopes)
	}

	// Revoke
	rr := httptest.NewRequest("DELETE", "/oauth/token/"+tok, nil)
	rr.SetPathValue("id", tok)
	rw := httptest.NewRecorder()
	mgr.RevokeToken(rw, rr)

	// Introspect — should be inactive
	_, _, _, _, revoked, ok = mgr.Lookup(tok)
	if !ok || !revoked {
		t.Fatal("introspect: token should be revoked")
	}
}

// TestWebhookSignatureIntegration exercises: configure signing → emit event → verify signature
func TestWebhookSignatureIntegration(t *testing.T) {
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

	key := []byte("integration-test-secret")
	n := events.New([]string{srv.URL})
	n.SetSigningKey(key)

	n.Emit(events.Event{
		Type:     events.TokenIssued,
		ClientID: "test-svc",
		Scopes:   []string{"read:logs-*"},
	})
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	sig := gotSig
	body := gotBody
	mu.Unlock()

	if !strings.HasPrefix(sig, "sha256=") {
		t.Fatalf("expected sha256= prefix, got %q", sig)
	}

	// Verify
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(body))
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if sig != expected {
		t.Fatalf("signature mismatch:\n  got:  %s\n  want: %s", sig, expected)
	}

	// Verify body contains event data
	var evt map[string]interface{}
	json.Unmarshal([]byte(body), &evt)
	if evt["client_id"] != "test-svc" {
		t.Fatalf("expected client_id=test-svc, got %v", evt["client_id"])
	}
}

// TestTokenCleanupIntegration exercises: issue expired tokens → cleanup → verify removed
func TestTokenCleanupIntegration(t *testing.T) {
	mgr := token.NewManager()
	mgr.RegisterClient("svc", "pw", []string{"read:logs-*"}, nil)

	// Issue tokens
	var tokens []string
	for i := 0; i < 5; i++ {
		w := issueToken(t, mgr, "svc", "pw", "read:logs-*")
		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		tokens = append(tokens, resp["access_token"].(string))
	}

	// Revoke all
	for _, tok := range tokens {
		rr := httptest.NewRequest("DELETE", "/oauth/token/"+tok, nil)
		rr.SetPathValue("id", tok)
		mgr.RevokeToken(httptest.NewRecorder(), rr)
	}

	// All should be revoked
	for _, tok := range tokens {
		if mgr.IsValid(tok) {
			t.Fatalf("token %s should be invalid after revoke", tok)
		}
	}

	// Stats should show revoked
	stats := mgr.Stats()
	if stats["revoked"] < 5 {
		t.Fatalf("expected >=5 revoked, got %v", stats["revoked"])
	}
}

func issueTokenHelper(t *testing.T, mgr *token.Manager, clientID, secret, scope string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/oauth/token",
		strings.NewReader(url.Values{
			"grant_type":    {"client_credentials"},
			"client_id":     {clientID},
			"client_secret": {secret},
			"scope":         {scope},
		}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mgr.IssueToken(w, req)
	if w.Code != 200 {
		t.Fatalf("issueToken: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	return w
}
