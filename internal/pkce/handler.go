// Package pkce implements Proof Key for Code Exchange (RFC 7636) for browser clients.
package pkce

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// AuthCode is a pending authorization code.
type AuthCode struct {
	Code                string
	ClientID            string
	Scopes              []string
	CodeChallenge       string
	CodeChallengeMethod string
	CreatedAt           time.Time
	RedirectURI         string
}

// Handler manages PKCE authorization code flow.
type Handler struct {
	codes            map[string]*AuthCode
	pending          map[string]*pendingConsent
	mu               sync.Mutex
	issuer           func(clientID string, scopes []string) (accessToken, refreshToken string)
	validateRedirect func(clientID, uri string) bool
}

// NewHandler creates a PKCE handler. issuer mints tokens; validateRedirect checks redirect_uri allowlist.
func NewHandler(issuer func(clientID string, scopes []string) (string, string), validateRedirect func(clientID, uri string) bool) *Handler {
	return &Handler{
		codes:            make(map[string]*AuthCode),
		issuer:           issuer,
		validateRedirect: validateRedirect,
	}
}

// pendingConsent stores an authorize request awaiting user approval.
type pendingConsent struct {
	ID                  string
	ClientID            string
	Scopes              []string
	CodeChallenge       string
	CodeChallengeMethod string
	RedirectURI         string
	State               string
	CreatedAt           time.Time
}

// Authorize handles GET /oauth/authorize — shows consent screen.
func (h *Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	challenge := r.URL.Query().Get("code_challenge")
	method := r.URL.Query().Get("code_challenge_method")
	redirectURI := r.URL.Query().Get("redirect_uri")
	scope := r.URL.Query().Get("scope")
	state := r.URL.Query().Get("state")

	if clientID == "" || challenge == "" || redirectURI == "" {
		writeErr(w, http.StatusBadRequest, "invalid_request", "client_id, code_challenge, redirect_uri required")
		return
	}
	if h.validateRedirect != nil && !h.validateRedirect(clientID, redirectURI) {
		writeErr(w, http.StatusBadRequest, "invalid_request", "redirect_uri not registered for client")
		return
	}
	if method == "" {
		method = "S256"
	}
	if method != "S256" {
		writeErr(w, http.StatusBadRequest, "invalid_request", "only S256 code_challenge_method supported")
		return
	}

	consentID := generateCode()
	h.mu.Lock()
	if h.pending == nil {
		h.pending = make(map[string]*pendingConsent)
	}
	h.pending[consentID] = &pendingConsent{
		ID: consentID, ClientID: clientID, Scopes: splitScopes(scope),
		CodeChallenge: challenge, CodeChallengeMethod: method,
		RedirectURI: redirectURI, State: state, CreatedAt: time.Now(),
	}
	h.mu.Unlock()

	renderConsent(w, consentID, clientID, splitScopes(scope))
}

// Consent handles POST /oauth/consent — approve or deny.
func (h *Handler) Consent(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	consentID := r.FormValue("consent_id")
	action := r.FormValue("action")

	h.mu.Lock()
	pc, ok := h.pending[consentID]
	if ok {
		delete(h.pending, consentID)
	}
	h.mu.Unlock()

	if !ok || time.Since(pc.CreatedAt) > 10*time.Minute {
		writeErr(w, http.StatusBadRequest, "invalid_request", "consent expired or not found")
		return
	}

	sep := "?"
	if len(pc.RedirectURI) > 0 && pc.RedirectURI[len(pc.RedirectURI)-1] == '?' {
		sep = ""
	}
	stateParam := ""
	if pc.State != "" {
		stateParam = "&state=" + pc.State
	}

	if action != "approve" {
		http.Redirect(w, r, fmt.Sprintf("%s%serror=access_denied%s", pc.RedirectURI, sep, stateParam), http.StatusFound)
		return
	}

	code := generateCode()
	h.mu.Lock()
	h.codes[code] = &AuthCode{
		Code: code, ClientID: pc.ClientID, Scopes: pc.Scopes,
		CodeChallenge: pc.CodeChallenge, CodeChallengeMethod: pc.CodeChallengeMethod,
		CreatedAt: time.Now(), RedirectURI: pc.RedirectURI,
	}
	h.mu.Unlock()

	http.Redirect(w, r, fmt.Sprintf("%s%scode=%s%s", pc.RedirectURI, sep, code, stateParam), http.StatusFound)
}

// Exchange handles POST /oauth/token with grant_type=authorization_code.
func (h *Handler) Exchange(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	code := r.FormValue("code")
	verifier := r.FormValue("code_verifier")
	redirectURI := r.FormValue("redirect_uri")

	if code == "" || verifier == "" {
		writeErr(w, http.StatusBadRequest, "invalid_request", "code and code_verifier required")
		return
	}

	h.mu.Lock()
	ac, ok := h.codes[code]
	if ok {
		delete(h.codes, code) // one-time use
	}
	h.mu.Unlock()

	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid_grant", "authorization code not found or already used")
		return
	}

	// Expire after 10 minutes
	if time.Since(ac.CreatedAt) > 10*time.Minute {
		writeErr(w, http.StatusBadRequest, "invalid_grant", "authorization code expired")
		return
	}

	if ac.RedirectURI != redirectURI {
		writeErr(w, http.StatusBadRequest, "invalid_grant", "redirect_uri mismatch")
		return
	}

	// Verify PKCE: SHA256(code_verifier) must match code_challenge
	hash := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])
	if subtle.ConstantTimeCompare([]byte(computed), []byte(ac.CodeChallenge)) != 1 {
		writeErr(w, http.StatusBadRequest, "invalid_grant", "code_verifier failed verification")
		return
	}

	accessToken, refreshToken := h.issuer(ac.ClientID, ac.Scopes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": refreshToken,
	})
}

// Cleanup removes expired codes. Call periodically.
func (h *Handler) Cleanup() {
	h.mu.Lock()
	for k, v := range h.codes {
		if time.Since(v.CreatedAt) > 10*time.Minute {
			delete(h.codes, k)
		}
	}
	h.mu.Unlock()
}

func generateCode() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func splitScopes(s string) []string {
	if s == "" {
		return nil
	}
	var scopes []string
	for _, p := range split(s) {
		if p != "" {
			scopes = append(scopes, p)
		}
	}
	return scopes
}

func split(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func writeErr(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": code, "error_description": desc})
}
