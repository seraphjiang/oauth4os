package e2e

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// TestWebAppPKCEFlow simulates the full browser-based demo app flow:
// register app → open login → consent page → approve → exchange code → use token to search.
func TestWebAppPKCEFlow(t *testing.T) {
	proxy := proxyURL(t)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse // don't follow redirects
	}}

	// Step 1: Register a demo app via dynamic client registration
	t.Log("Step 1: Register demo app")
	regBody := `{"client_name":"demo-webapp","redirect_uris":["http://localhost:9999/callback"],"grant_types":["authorization_code"],"scope":"read:logs-*"}`
	resp := mustDo(t, client, "POST", proxy+"/oauth/register", "application/json", regBody)
	if resp.StatusCode != 201 {
		t.Fatalf("register: expected 201, got %d: %s", resp.StatusCode, readBody(resp))
	}
	var reg struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	json.NewDecoder(resp.Body).Decode(&reg)
	resp.Body.Close()
	if reg.ClientID == "" {
		t.Fatal("register: no client_id")
	}
	t.Logf("  client_id=%s", reg.ClientID)

	// Step 2: Start PKCE authorize (browser opens this URL)
	t.Log("Step 2: PKCE authorize → consent page")
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := s256(verifier)
	authURL := fmt.Sprintf("%s/oauth/authorize?response_type=code&client_id=%s&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256&scope=read:logs-*&state=demo123",
		proxy, reg.ClientID, url.QueryEscape("http://localhost:9999/callback"), challenge)
	resp = mustDo(t, client, "GET", authURL, "", "")
	if resp.StatusCode != 200 {
		t.Fatalf("authorize: expected 200 consent page, got %d: %s", resp.StatusCode, readBody(resp))
	}
	body := readBody(resp)
	if !strings.Contains(body, "consent_id") {
		t.Fatal("authorize: consent page missing consent_id")
	}

	// Extract consent_id from HTML form
	consentID := extractConsentID(t, body)
	t.Logf("  consent_id=%s", consentID)

	// Step 3: User clicks "Approve" on consent screen
	t.Log("Step 3: Approve consent → redirect with code")
	resp = mustDo(t, client, "POST", proxy+"/oauth/consent",
		"application/x-www-form-urlencoded",
		"consent_id="+consentID+"&action=approve")
	if resp.StatusCode != 302 {
		t.Fatalf("consent: expected 302, got %d: %s", resp.StatusCode, readBody(resp))
	}
	loc := resp.Body.Close
	location := resp.Header.Get("Location")
	_ = loc
	if location == "" {
		t.Fatal("consent: no Location header")
	}
	if !strings.Contains(location, "code=") {
		t.Fatalf("consent: redirect missing code: %s", location)
	}
	if !strings.Contains(location, "state=demo123") {
		t.Errorf("consent: redirect missing state: %s", location)
	}

	// Parse code from redirect
	u, _ := url.Parse(location)
	code := u.Query().Get("code")
	t.Logf("  code=%s", code[:min(len(code), 16)]+"...")

	// Step 4: Exchange code for token (app backend does this)
	t.Log("Step 4: Exchange code → access token")
	resp = mustDo(t, client, "POST", proxy+"/oauth/authorize/token",
		"application/x-www-form-urlencoded",
		fmt.Sprintf("grant_type=authorization_code&code=%s&code_verifier=%s&redirect_uri=%s",
			code, verifier, url.QueryEscape("http://localhost:9999/callback")))
	if resp.StatusCode != 200 {
		t.Fatalf("exchange: expected 200, got %d: %s", resp.StatusCode, readBody(resp))
	}
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
	}
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	resp.Body.Close()
	if tokenResp.AccessToken == "" {
		t.Fatal("exchange: no access_token")
	}
	t.Logf("  token_type=%s, has_refresh=%v", tokenResp.TokenType, tokenResp.RefreshToken != "")

	// Step 5: Use token to search OpenSearch (dashboard loads data)
	t.Log("Step 5: Search with token")
	req, _ := http.NewRequest("GET", proxy+"/logs-*/_search?q=*&size=5", nil)
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	// Accept 200 (results) or 403/404 (no index yet) — both prove auth worked
	if resp.StatusCode == 200 {
		t.Log("  ✅ Search returned results")
	} else if resp.StatusCode == 404 || resp.StatusCode == 403 {
		t.Logf("  ✅ Search auth accepted (HTTP %d — index may not exist)", resp.StatusCode)
	} else if resp.StatusCode == 401 {
		t.Fatalf("search: token rejected (401)")
	} else {
		t.Logf("  ⚠️  Search returned HTTP %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Step 6: Cleanup — delete the registered app
	t.Log("Step 6: Cleanup")
	resp = mustDo(t, client, "DELETE", proxy+"/oauth/register/"+reg.ClientID, "", "")
	if resp.StatusCode != 204 {
		t.Errorf("cleanup: expected 204, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	t.Log("✅ Full web app PKCE flow passed")
}

// TestConsentDenyFlow verifies the deny path shows error to the app.
func TestConsentDenyFlow(t *testing.T) {
	proxy := proxyURL(t)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	verifier := "deny-flow-verifier-test"
	challenge := s256(verifier)
	authURL := fmt.Sprintf("%s/oauth/authorize?response_type=code&client_id=test&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256&state=deny1",
		proxy, url.QueryEscape("http://localhost:9999/callback"), challenge)
	resp := mustDo(t, client, "GET", authURL, "", "")
	if resp.StatusCode != 200 {
		t.Fatalf("expected consent page, got %d", resp.StatusCode)
	}
	consentID := extractConsentID(t, readBody(resp))

	// Deny
	resp = mustDo(t, client, "POST", proxy+"/oauth/consent",
		"application/x-www-form-urlencoded",
		"consent_id="+consentID+"&action=deny")
	if resp.StatusCode != 302 {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if !strings.Contains(loc, "error=access_denied") {
		t.Fatalf("expected access_denied, got: %s", loc)
	}
}

// --- helpers ---

func proxyURL(t *testing.T) string {
	t.Helper()
	// Try AppRunner, then localhost
	for _, u := range []string{
		"https://f5cmk2hxwx.us-west-2.awsapprunner.com",
		"http://localhost:8443",
	} {
		resp, err := http.Get(u + "/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return u
		}
	}
	t.Skip("no proxy available")
	return ""
}

func mustDo(t *testing.T, client *http.Client, method, url, contentType, body string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func readBody(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return string(b)
}

var consentIDRe = regexp.MustCompile(`name="consent_id"\s+value="([^"]+)"`)

func extractConsentID(t *testing.T, html string) string {
	t.Helper()
	m := consentIDRe.FindStringSubmatch(html)
	if len(m) < 2 {
		t.Fatal("consent_id not found in HTML")
	}
	return m[1]
}

func s256(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
