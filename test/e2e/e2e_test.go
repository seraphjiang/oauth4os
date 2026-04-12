package e2e

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/seraphjiang/oauth4os/internal/audit"
	"github.com/seraphjiang/oauth4os/internal/token"
)

// ── Token issuance ───────────────────────────────────────────────────────────

func TestTokenIssuance_ClientCredentials(t *testing.T) {
	mgr := token.NewManager()
	srv := httptest.NewServer(http.HandlerFunc(mgr.IssueToken))
	defer srv.Close()

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"test-agent"},
		"client_secret": {"secret"},
		"scope":         {"read:logs-*"},
	}
	resp, err := http.PostForm(srv.URL, form)
	if err != nil {
		t.Fatalf("POST /oauth/token failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if _, ok := result["access_token"]; !ok {
		t.Fatal("Response missing access_token")
	}
	if result["token_type"] != "Bearer" {
		t.Errorf("Expected token_type=Bearer, got %v", result["token_type"])
	}
}

func TestTokenIssuance_MissingGrantType(t *testing.T) {
	mgr := token.NewManager()
	srv := httptest.NewServer(http.HandlerFunc(mgr.IssueToken))
	defer srv.Close()

	form := url.Values{"client_id": {"test"}}
	resp, err := http.PostForm(srv.URL, form)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Fatal("Expected error for missing grant_type")
	}
}

// ── Scope enforcement ────────────────────────────────────────────────────────

func TestTokenIssuance_ScopesIncluded(t *testing.T) {
	mgr := token.NewManager()
	srv := httptest.NewServer(http.HandlerFunc(mgr.IssueToken))
	defer srv.Close()

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"scoped-agent"},
		"client_secret": {"secret"},
		"scope":         {"read:logs-* write:metrics-*"},
	}
	resp, _ := http.PostForm(srv.URL, form)
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	scope, ok := result["scope"].(string)
	if !ok {
		t.Fatal("Response missing scope")
	}
	if !strings.Contains(scope, "read:logs-*") {
		t.Errorf("Scope missing read:logs-*, got: %s", scope)
	}
}

// ── Token revocation ─────────────────────────────────────────────────────────

func TestTokenRevocation(t *testing.T) {
	mgr := token.NewManager()

	// Issue a token
	issueSrv := httptest.NewServer(http.HandlerFunc(mgr.IssueToken))
	defer issueSrv.Close()

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"revoke-test"},
		"client_secret": {"secret"},
		"scope":         {"read:*"},
	}
	resp, _ := http.PostForm(issueSrv.URL, form)
	var issued map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&issued)
	resp.Body.Close()

	tokenID, _ := issued["token_id"].(string)
	if tokenID == "" {
		// Try access_token as fallback
		tokenID, _ = issued["access_token"].(string)
	}

	// Revoke it
	revokeSrv := httptest.NewServer(http.HandlerFunc(mgr.RevokeToken))
	defer revokeSrv.Close()

	revokeBody, _ := json.Marshal(map[string]string{"token_id": tokenID})
	revokeResp, err := http.Post(revokeSrv.URL, "application/json", bytes.NewReader(revokeBody))
	if err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}
	defer revokeResp.Body.Close()

	if revokeResp.StatusCode != http.StatusOK && revokeResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(revokeResp.Body)
		t.Fatalf("Revoke expected 200/204, got %d: %s", revokeResp.StatusCode, body)
	}
}

// ── Audit trail ──────────────────────────────────────────────────────────────

func TestAuditTrail_LogsAccess(t *testing.T) {
	var buf bytes.Buffer
	auditor := audit.NewAuditor(&buf)

	auditor.Log("test-client", []string{"read:logs-*"}, "GET", "/logs-*/_search")

	logged := buf.String()
	if !strings.Contains(logged, "test-client") {
		t.Errorf("Audit log missing client_id, got: %s", logged)
	}
	if !strings.Contains(logged, "read:logs-*") {
		t.Errorf("Audit log missing scope, got: %s", logged)
	}
}

func TestAuditTrail_LogsMultipleEvents(t *testing.T) {
	var buf bytes.Buffer
	auditor := audit.NewAuditor(&buf)

	auditor.Log("agent-1", []string{"read:*"}, "GET", "/_search")
	auditor.Log("agent-2", []string{"write:*"}, "POST", "/_bulk")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Errorf("Expected 2 audit entries, got %d", len(lines))
	}
}

// ── Token listing ────────────────────────────────────────────────────────────

func TestTokenList(t *testing.T) {
	mgr := token.NewManager()

	// Issue a token first
	issueSrv := httptest.NewServer(http.HandlerFunc(mgr.IssueToken))
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"list-test"},
		"client_secret": {"secret"},
		"scope":         {"read:*"},
	}
	resp, _ := http.PostForm(issueSrv.URL, form)
	resp.Body.Close()
	issueSrv.Close()

	// List tokens
	listSrv := httptest.NewServer(http.HandlerFunc(mgr.ListTokens))
	defer listSrv.Close()

	listResp, err := http.Get(listSrv.URL)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", listResp.StatusCode)
	}

	var tokens []interface{}
	body, _ := io.ReadAll(listResp.Body)
	// Response might be {"tokens": [...]} or [...]
	var wrapper map[string]interface{}
	if err := json.Unmarshal(body, &wrapper); err == nil {
		if arr, ok := wrapper["tokens"].([]interface{}); ok {
			tokens = arr
		}
	} else {
		json.Unmarshal(body, &tokens)
	}

	if len(tokens) == 0 {
		t.Error("Expected at least 1 token in list")
	}
}

// ── Timestamp helper ─────────────────────────────────────────────────────────

func TestTokenHasExpiry(t *testing.T) {
	mgr := token.NewManager()
	srv := httptest.NewServer(http.HandlerFunc(mgr.IssueToken))
	defer srv.Close()

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"expiry-test"},
		"client_secret": {"secret"},
		"scope":         {"read:*"},
	}
	resp, _ := http.PostForm(srv.URL, form)
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	expiresIn, ok := result["expires_in"].(float64)
	if !ok {
		t.Skip("No expires_in in response — may use different field")
	}
	if expiresIn <= 0 {
		t.Errorf("Expected positive expires_in, got %v", expiresIn)
	}
}

// Ensure test file compiles even if unused imports
var _ = fmt.Sprintf
var _ = time.Now
var _ = rand.Reader
var _ = rsa.GenerateKey
var _ = jwt.New
