package integration

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/pkce"
)

// TestPKCEAuthCodeFullFlow exercises: authorize → consent → exchange with PKCE
func TestPKCEAuthCodeFullFlow(t *testing.T) {
	h := pkce.NewHandler(
		func(clientID string, scopes []string) (string, string) { return "tok_pkce", "rtk_pkce" },
		func(clientID, redirectURI string) bool { return true },
	)

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	// Step 1: Authorize
	authURL := "/oauth/authorize?client_id=app&redirect_uri=https://app/cb&scope=read:logs-*&state=xyz&response_type=code&code_challenge=" + challenge + "&code_challenge_method=S256"
	w := httptest.NewRecorder()
	h.Authorize(w, httptest.NewRequest("GET", authURL, nil))

	if w.Code != 200 {
		t.Fatalf("authorize: expected 200, got %d", w.Code)
	}

	// Extract consent_id from HTML form
	re := regexp.MustCompile(`name="consent_id"\s+value="([^"]+)"`)
	matches := re.FindStringSubmatch(w.Body.String())
	if len(matches) < 2 {
		t.Fatal("authorize: consent_id not found in response")
	}
	consentID := matches[1]

	// Step 2: Approve consent
	w2 := httptest.NewRecorder()
	consentReq := httptest.NewRequest("POST", "/oauth/consent",
		strings.NewReader(url.Values{
			"consent_id": {consentID},
			"action":     {"approve"},
		}.Encode()))
	consentReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.Consent(w2, consentReq)

	if w2.Code != http.StatusFound {
		t.Fatalf("consent: expected 302, got %d: %s", w2.Code, w2.Body.String())
	}

	// Extract code from redirect
	loc := w2.Header().Get("Location")
	u, _ := url.Parse(loc)
	code := u.Query().Get("code")
	if code == "" {
		t.Fatalf("consent: no code in redirect: %s", loc)
	}
	if u.Query().Get("state") != "xyz" {
		t.Fatalf("consent: state mismatch: %s", loc)
	}

	// Step 3: Exchange code for token
	w3 := httptest.NewRecorder()
	exchangeReq := httptest.NewRequest("POST", "/oauth/token",
		strings.NewReader(url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"code_verifier": {verifier},
			"redirect_uri":  {"https://app/cb"},
		}.Encode()))
	exchangeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.Exchange(w3, exchangeReq)

	if w3.Code != 200 {
		t.Fatalf("exchange: expected 200, got %d: %s", w3.Code, w3.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w3.Body).Decode(&resp)
	if resp["access_token"] != "tok_pkce" {
		t.Fatalf("exchange: expected tok_pkce, got %v", resp["access_token"])
	}
}

// TestAuthCodeWithoutPKCEFullFlow exercises: authorize → consent → exchange without PKCE
func TestAuthCodeWithoutPKCEFullFlow(t *testing.T) {
	h := pkce.NewHandler(
		func(clientID string, scopes []string) (string, string) { return "tok_nopkce", "rtk_nopkce" },
		func(clientID, redirectURI string) bool { return true },
	)

	// Authorize without code_challenge
	w := httptest.NewRecorder()
	h.Authorize(w, httptest.NewRequest("GET", "/oauth/authorize?client_id=svc&redirect_uri=https://app/cb&scope=read:logs-*&response_type=code", nil))

	if w.Code != 200 {
		t.Fatalf("authorize: expected 200, got %d", w.Code)
	}

	re := regexp.MustCompile(`name="consent_id"\s+value="([^"]+)"`)
	matches := re.FindStringSubmatch(w.Body.String())
	if len(matches) < 2 {
		t.Fatal("consent_id not found")
	}

	// Approve
	w2 := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/oauth/consent",
		strings.NewReader(url.Values{"consent_id": {matches[1]}, "action": {"approve"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.Consent(w2, req)

	loc := w2.Header().Get("Location")
	u, _ := url.Parse(loc)
	code := u.Query().Get("code")

	// Exchange without code_verifier — should work
	w3 := httptest.NewRecorder()
	exReq := httptest.NewRequest("POST", "/oauth/token",
		strings.NewReader(url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {code},
			"redirect_uri": {"https://app/cb"},
		}.Encode()))
	exReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.Exchange(w3, exReq)

	if w3.Code != 200 {
		t.Fatalf("exchange without PKCE: expected 200, got %d: %s", w3.Code, w3.Body.String())
	}
}
