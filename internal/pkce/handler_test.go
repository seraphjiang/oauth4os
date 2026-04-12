package pkce

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPKCEFlow(t *testing.T) {
	var issuedClient string
	h := NewHandler(func(clientID string, scopes []string) (string, string) {
		issuedClient = clientID
		return "tok_abc", "rtk_abc"
	})

	// Generate PKCE pair
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	// Step 1: Authorize
	authURL := "/oauth/authorize?client_id=myapp&code_challenge=" + challenge +
		"&code_challenge_method=S256&redirect_uri=http://localhost/callback&scope=read:logs-*"
	req := httptest.NewRequest(http.MethodGet, authURL, nil)
	w := httptest.NewRecorder()
	h.Authorize(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	u, _ := url.Parse(loc)
	code := u.Query().Get("code")
	if code == "" {
		t.Fatal("no code in redirect")
	}

	// Step 2: Exchange
	form := url.Values{
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {"http://localhost/callback"},
	}
	req = httptest.NewRequest(http.MethodPost, "/oauth/authorize/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h.Exchange(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["access_token"] != "tok_abc" {
		t.Fatalf("expected tok_abc, got %v", resp["access_token"])
	}
	if issuedClient != "myapp" {
		t.Fatalf("expected myapp, got %s", issuedClient)
	}
}

func TestPKCEBadVerifier(t *testing.T) {
	h := NewHandler(func(clientID string, scopes []string) (string, string) {
		return "tok", "rtk"
	})

	hash := sha256.Sum256([]byte("correct-verifier"))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=app&code_challenge="+challenge+
		"&code_challenge_method=S256&redirect_uri=http://localhost/cb", nil)
	w := httptest.NewRecorder()
	h.Authorize(w, req)
	loc := w.Header().Get("Location")
	u, _ := url.Parse(loc)
	code := u.Query().Get("code")

	// Exchange with wrong verifier
	form := url.Values{
		"code":          {code},
		"code_verifier": {"wrong-verifier"},
		"redirect_uri":  {"http://localhost/cb"},
	}
	req = httptest.NewRequest(http.MethodPost, "/oauth/authorize/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h.Exchange(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPKCECodeReuse(t *testing.T) {
	h := NewHandler(func(clientID string, scopes []string) (string, string) {
		return "tok", "rtk"
	})

	verifier := "test-verifier-string"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=app&code_challenge="+challenge+
		"&code_challenge_method=S256&redirect_uri=http://localhost/cb", nil)
	w := httptest.NewRecorder()
	h.Authorize(w, req)
	u, _ := url.Parse(w.Header().Get("Location"))
	code := u.Query().Get("code")

	// First exchange — should succeed
	form := url.Values{"code": {code}, "code_verifier": {verifier}, "redirect_uri": {"http://localhost/cb"}}
	req = httptest.NewRequest(http.MethodPost, "/oauth/authorize/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h.Exchange(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first exchange: expected 200, got %d", w.Code)
	}

	// Second exchange — code already consumed
	req = httptest.NewRequest(http.MethodPost, "/oauth/authorize/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h.Exchange(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("second exchange: expected 400, got %d", w.Code)
	}
}
