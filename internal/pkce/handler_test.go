package pkce

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func makeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func testHandler() *Handler {
	return NewHandler(func(clientID string, scopes []string) (string, string) {
		return "tok_test", "rtk_test"
	})
}

func TestAuthorize_Success(t *testing.T) {
	h := testHandler()
	challenge := makeChallenge("my-verifier")
	r := httptest.NewRequest("GET", "/oauth/authorize?client_id=app&code_challenge="+challenge+"&code_challenge_method=S256&redirect_uri=http://localhost/cb&scope=read:logs", nil)
	w := httptest.NewRecorder()
	h.Authorize(w, r)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d: %s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, "http://localhost/cb?code=") {
		t.Errorf("unexpected redirect: %s", loc)
	}
}

func TestAuthorize_MissingParams(t *testing.T) {
	h := testHandler()
	r := httptest.NewRequest("GET", "/oauth/authorize?client_id=app", nil)
	w := httptest.NewRecorder()
	h.Authorize(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAuthorize_UnsupportedMethod(t *testing.T) {
	h := testHandler()
	r := httptest.NewRequest("GET", "/oauth/authorize?client_id=app&code_challenge=x&code_challenge_method=plain&redirect_uri=http://localhost/cb", nil)
	w := httptest.NewRecorder()
	h.Authorize(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for plain method, got %d", w.Code)
	}
}

func TestExchange_Success(t *testing.T) {
	h := testHandler()
	verifier := "my-verifier-string-for-pkce-test"
	challenge := makeChallenge(verifier)

	// Authorize
	ar := httptest.NewRequest("GET", "/oauth/authorize?client_id=app&code_challenge="+challenge+"&code_challenge_method=S256&redirect_uri=http://localhost/cb&scope=read:logs", nil)
	aw := httptest.NewRecorder()
	h.Authorize(aw, ar)
	loc := aw.Header().Get("Location")
	code := strings.TrimPrefix(loc, "http://localhost/cb?code=")

	// Exchange
	form := "grant_type=authorization_code&code=" + code + "&code_verifier=" + verifier + "&redirect_uri=http://localhost/cb"
	er := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form))
	er.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ew := httptest.NewRecorder()
	h.Exchange(ew, er)
	if ew.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", ew.Code, ew.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(ew.Body.Bytes(), &resp)
	if resp["access_token"] != "tok_test" {
		t.Errorf("unexpected token: %v", resp["access_token"])
	}
}

func TestExchange_BadVerifier(t *testing.T) {
	h := testHandler()
	challenge := makeChallenge("correct-verifier")

	ar := httptest.NewRequest("GET", "/oauth/authorize?client_id=app&code_challenge="+challenge+"&code_challenge_method=S256&redirect_uri=http://localhost/cb", nil)
	aw := httptest.NewRecorder()
	h.Authorize(aw, ar)
	code := strings.TrimPrefix(aw.Header().Get("Location"), "http://localhost/cb?code=")

	form := "code=" + code + "&code_verifier=wrong-verifier&redirect_uri=http://localhost/cb"
	er := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form))
	er.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ew := httptest.NewRecorder()
	h.Exchange(ew, er)
	if ew.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad verifier, got %d", ew.Code)
	}
}

func TestExchange_CodeReuse(t *testing.T) {
	h := testHandler()
	verifier := "reuse-test-verifier"
	challenge := makeChallenge(verifier)

	ar := httptest.NewRequest("GET", "/oauth/authorize?client_id=app&code_challenge="+challenge+"&code_challenge_method=S256&redirect_uri=http://localhost/cb", nil)
	aw := httptest.NewRecorder()
	h.Authorize(aw, ar)
	code := strings.TrimPrefix(aw.Header().Get("Location"), "http://localhost/cb?code=")

	// First exchange succeeds
	form := "code=" + code + "&code_verifier=" + verifier + "&redirect_uri=http://localhost/cb"
	er := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form))
	er.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ew := httptest.NewRecorder()
	h.Exchange(ew, er)
	if ew.Code != 200 {
		t.Fatalf("first exchange should succeed, got %d", ew.Code)
	}

	// Second exchange fails (one-time use)
	er2 := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form))
	er2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ew2 := httptest.NewRecorder()
	h.Exchange(ew2, er2)
	if ew2.Code != http.StatusBadRequest {
		t.Fatalf("code reuse should fail, got %d", ew2.Code)
	}
}

func TestExchange_RedirectMismatch(t *testing.T) {
	h := testHandler()
	verifier := "redirect-test"
	challenge := makeChallenge(verifier)

	ar := httptest.NewRequest("GET", "/oauth/authorize?client_id=app&code_challenge="+challenge+"&code_challenge_method=S256&redirect_uri=http://localhost/cb", nil)
	aw := httptest.NewRecorder()
	h.Authorize(aw, ar)
	code := strings.TrimPrefix(aw.Header().Get("Location"), "http://localhost/cb?code=")

	form := "code=" + code + "&code_verifier=" + verifier + "&redirect_uri=http://evil.com/cb"
	er := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form))
	er.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ew := httptest.NewRecorder()
	h.Exchange(ew, er)
	if ew.Code != http.StatusBadRequest {
		t.Fatalf("redirect mismatch should fail, got %d", ew.Code)
	}
}

func TestExchange_MissingParams(t *testing.T) {
	h := testHandler()
	er := httptest.NewRequest("POST", "/oauth/token", strings.NewReader("code=&code_verifier="))
	er.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ew := httptest.NewRecorder()
	h.Exchange(ew, er)
	if ew.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", ew.Code)
	}
}

func TestCleanup(t *testing.T) {
	h := testHandler()
	h.mu.Lock()
	h.codes["old"] = &AuthCode{Code: "old", CreatedAt: time.Now().Add(-20 * time.Minute)}
	h.codes["fresh"] = &AuthCode{Code: "fresh", CreatedAt: time.Now()}
	h.mu.Unlock()
	h.Cleanup()
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.codes["old"]; ok {
		t.Error("expired code should be cleaned up")
	}
	if _, ok := h.codes["fresh"]; !ok {
		t.Error("fresh code should survive cleanup")
	}
}
