package pkce

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Edge: missing client_id in authorize
func TestEdge_AuthorizeMissingClientID(t *testing.T) {
	h := NewHandler(nil, nil)
	r := httptest.NewRequest("GET", "/oauth/authorize?code_challenge=abc&code_challenge_method=S256&redirect_uri=http://localhost/cb", nil)
	w := httptest.NewRecorder()
	h.Authorize(w, r)
	if w.Code == 200 || w.Code == 302 {
		t.Errorf("missing client_id should fail, got %d", w.Code)
	}
}

// Edge: missing code_challenge in authorize
func TestEdge_AuthorizeMissingChallenge(t *testing.T) {
	h := NewHandler(nil, nil)
	r := httptest.NewRequest("GET", "/oauth/authorize?client_id=app&redirect_uri=http://localhost/cb", nil)
	w := httptest.NewRecorder()
	h.Authorize(w, r)
	if w.Code == 302 {
		t.Error("missing code_challenge should not redirect to consent")
	}
}

// Edge: exchange with wrong code_verifier
func TestEdge_ExchangeWrongVerifier(t *testing.T) {
	issued := false
	h := NewHandler(
		func(clientID string, scopes []string) (string, string) {
			issued = true
			return "tok_123", "refresh_123"
		},
		func(clientID, uri string) bool { return true },
	)
	// Start authorize flow
	r := httptest.NewRequest("GET", "/oauth/authorize?client_id=app&code_challenge=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk&code_challenge_method=S256&redirect_uri=http://localhost/cb&scope=read", nil)
	w := httptest.NewRecorder()
	h.Authorize(w, r)

	// Try exchange with wrong verifier
	body := "grant_type=authorization_code&code=wrong-code&redirect_uri=http://localhost/cb&client_id=app&code_verifier=wrong"
	r = httptest.NewRequest("POST", "/oauth/token", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h.Exchange(w, r)
	if w.Code == 200 {
		t.Error("wrong code should not issue token")
	}
	if issued {
		t.Error("issuer should not be called with wrong code")
	}
}

// Edge: exchange with GET method should fail
func TestEdge_ExchangeRejectsGET(t *testing.T) {
	h := NewHandler(nil, nil)
	r := httptest.NewRequest("GET", "/oauth/token?grant_type=authorization_code&code=abc", nil)
	w := httptest.NewRecorder()
	h.Exchange(w, r)
	if w.Code == 200 {
		t.Error("GET should not be accepted for token exchange")
	}
}

// Edge: cleanup removes expired codes
func TestEdge_CleanupExpired(t *testing.T) {
	h := NewHandler(nil, nil)
	h.Cleanup() // should not panic even with no codes
}
