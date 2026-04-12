package token

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// KeyProvider returns the current signing key.
type KeyProvider interface {
	CurrentKey() (kid string, key *rsa.PrivateKey)
}

// Token represents an issued access token.
type Token struct {
	ID           string    `json:"id"`
	ClientID     string    `json:"client_id"`
	Scopes       []string  `json:"scopes"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	Revoked      bool      `json:"revoked"`
	RefreshToken string    `json:"-"` // never exposed in list/get
}

// Client represents a registered OAuth client.
type Client struct {
	ID           string
	Secret       string
	Scopes       []string // allowed scopes
	RedirectURIs []string // allowed redirect URIs (required for PKCE)
}

const defaultTokenExpirySeconds = 3600 // 1 hour

// Manager handles token lifecycle.
type Manager struct {
	tokens      map[string]*Token
	refresh     map[string]string // refresh_token -> token_id
	usedRefresh map[string]string // used refresh_token -> client_id (reuse detection)
	families    map[string][]string // client_id -> [token_ids] (for family revocation)
	clients     map[string]*Client
	mu          sync.RWMutex
	jwtEnabled  bool
	issuer      string
	keyProvider KeyProvider
}

// NewManager creates a token manager.
func NewManager() *Manager {
	return &Manager{
		tokens:      make(map[string]*Token),
		refresh:     make(map[string]string),
		usedRefresh: make(map[string]string),
		families:    make(map[string][]string),
		clients:     make(map[string]*Client),
	}
}

// EnableJWT configures the manager to issue signed JWT access tokens.
func (m *Manager) EnableJWT(issuer string, kp KeyProvider) {
	m.jwtEnabled = true
	m.issuer = issuer
	m.keyProvider = kp
}

// RegisterClient adds a client for authentication.
func (m *Manager) RegisterClient(id, secret string, scopes, redirectURIs []string) {
	m.mu.Lock()
	m.clients[id] = &Client{ID: id, Secret: secret, Scopes: scopes, RedirectURIs: redirectURIs}
	m.mu.Unlock()
}

// Clients returns a snapshot of all registered clients.
func (m *Manager) Clients() []*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Client, 0, len(m.clients))
	for _, c := range m.clients {
		out = append(out, c)
	}
	return out
}

// ValidateRedirectURI checks if a redirect URI is allowed for a client.
func (m *Manager) ValidateRedirectURI(clientID, uri string) bool {
	m.mu.RLock()
	client, ok := m.clients[clientID]
	m.mu.RUnlock()
	if !ok || len(client.RedirectURIs) == 0 {
		return false
	}
	for _, allowed := range client.RedirectURIs {
		if allowed == uri {
			return true
		}
	}
	return false
}

// IsValid checks if a token ID is valid (not expired, not revoked).
func (m *Manager) IsValid(tokenID string) bool {
	m.mu.RLock()
	tok, ok := m.tokens[tokenID]
	if !ok {
		m.mu.RUnlock()
		return false
	}
	revoked := tok.Revoked
	expires := tok.ExpiresAt
	m.mu.RUnlock()
	return !revoked && time.Now().Before(expires)
}

// TouchToken extends a token's expiry if more than half its lifetime has elapsed (sliding window).
// Returns true if the token was extended.
func (m *Manager) TouchToken(tokenID string, window time.Duration) bool {
	if window <= 0 {
		window = 1 * time.Hour
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	tok, ok := m.tokens[tokenID]
	if !ok || tok.Revoked {
		return false
	}
	remaining := time.Until(tok.ExpiresAt)
	// Only extend if less than half the window remains
	if remaining < window/2 {
		tok.ExpiresAt = time.Now().Add(window)
		m.tokens[tokenID] = tok
		return true
	}
	return false
}

// IssueToken handles POST /oauth/token.
func (m *Manager) IssueToken(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	grantType := r.FormValue("grant_type")

	switch grantType {
	case "client_credentials":
		m.handleClientCredentials(w, r)
	case "refresh_token":
		m.handleRefreshToken(w, r)
	default:
		writeError(w, http.StatusBadRequest, "unsupported_grant_type", "use client_credentials or refresh_token")
	}
}

func (m *Manager) handleClientCredentials(w http.ResponseWriter, r *http.Request) {
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	scopeStr := r.FormValue("scope")

	// Support client_secret_basic (HTTP Basic Auth) per RFC 6749 §2.3.1
	if basicID, basicSecret, ok := r.BasicAuth(); ok && clientID == "" {
		clientID = basicID
		clientSecret = basicSecret
	}

	// Authenticate client
	if err := m.AuthenticateClient(clientID, clientSecret); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_client", "authentication failed")
		return
	}

	// Validate requested scopes
	scopes := strings.Fields(scopeStr)
	if len(scopes) == 0 {
		// Default to client's full registered scopes per OAuth 2.0 §3.3
		m.mu.RLock()
		if c, ok := m.clients[clientID]; ok && len(c.Scopes) > 0 {
			scopes = c.Scopes
		}
		m.mu.RUnlock()
	}
	if err := m.validateScopes(clientID, scopes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_scope", "requested scope exceeds client allowance")
		return
	}

	tok, refreshTok := m.createToken(clientID, scopes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  tok.ID,
		"token_type":    "Bearer",
		"expires_in":    defaultTokenExpirySeconds,
		"refresh_token": refreshTok,
		"scope":         strings.Join(scopes, " "),
	})
}

func (m *Manager) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.FormValue("refresh_token")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	// Support client_secret_basic
	if basicID, basicSecret, ok := r.BasicAuth(); ok && clientID == "" {
		clientID = basicID
		clientSecret = basicSecret
	}

	if err := m.AuthenticateClient(clientID, clientSecret); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_client", "authentication failed")
		return
	}

	m.mu.Lock()

	// Reuse detection: if this refresh token was already used, revoke entire family
	if stolenClient, reused := m.usedRefresh[refreshToken]; reused {
		m.revokeFamily(stolenClient)
		m.mu.Unlock()
		writeError(w, http.StatusBadRequest, "invalid_grant", "refresh token reuse detected — all tokens revoked")
		return
	}

	oldTokenID, ok := m.refresh[refreshToken]
	if !ok {
		m.mu.Unlock()
		writeError(w, http.StatusBadRequest, "invalid_grant", "refresh token not found or expired")
		return
	}
	oldToken, exists := m.tokens[oldTokenID]
	if !exists || oldToken.ClientID != clientID {
		m.mu.Unlock()
		writeError(w, http.StatusBadRequest, "invalid_grant", "refresh token does not belong to client")
		return
	}
	// Revoke old token + refresh token (rotation)
	oldToken.Revoked = true
	delete(m.refresh, refreshToken)
	m.usedRefresh[refreshToken] = clientID // track for reuse detection
	scopes := oldToken.Scopes
	m.mu.Unlock()

	// RFC 6749 §6: client may request narrower scope on refresh
	if requestedScope := r.FormValue("scope"); requestedScope != "" {
		requested := strings.Fields(requestedScope)
		allowed := make(map[string]bool, len(scopes))
		for _, s := range scopes {
			allowed[s] = true
		}
		for _, s := range requested {
			if !allowed[s] {
				writeError(w, http.StatusBadRequest, "invalid_scope", "requested scope exceeds original grant")
				return
			}
		}
		scopes = requested
	}

	tok, newRefresh := m.createToken(clientID, scopes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  tok.ID,
		"token_type":    "Bearer",
		"expires_in":    defaultTokenExpirySeconds,
		"refresh_token": newRefresh,
		"scope":         strings.Join(tok.Scopes, " "),
	})
}

func (m *Manager) createToken(clientID string, scopes []string) (*Token, string) {
	id := generateID("tok_")
	refreshTok := generateID("rtk_")

	tok := &Token{
		ID:           id,
		ClientID:     clientID,
		Scopes:       scopes,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		RefreshToken: refreshTok,
	}

	m.mu.Lock()
	m.tokens[id] = tok
	m.refresh[refreshTok] = id
	m.families[clientID] = append(m.families[clientID], id)
	m.mu.Unlock()

	return tok, refreshTok
}

// revokeFamily revokes all tokens for a client. Must be called with m.mu held.
func (m *Manager) revokeFamily(clientID string) {
	for _, tokID := range m.families[clientID] {
		if tok, ok := m.tokens[tokID]; ok {
			tok.Revoked = true
		}
	}
}

// CreateTokenForClient creates a token for a client (used by PKCE flow).
func (m *Manager) CreateTokenForClient(clientID string, scopes []string) (*Token, string) {
	return m.createToken(clientID, scopes)
}

// AuthenticateClient validates client credentials. Exported for PAR/revocation.
func (m *Manager) AuthenticateClient(clientID, secret string) error {
	m.mu.RLock()
	client, ok := m.clients[clientID]
	m.mu.RUnlock()
	if !ok {
		return errInvalidClient
	}
	if subtle.ConstantTimeCompare([]byte(client.Secret), []byte(secret)) != 1 {
		return errInvalidClient
	}
	return nil
}

func (m *Manager) validateScopes(clientID string, requested []string) error {
	m.mu.RLock()
	client, ok := m.clients[clientID]
	m.mu.RUnlock()
	if !ok {
		return errInvalidClient
	}
	if len(client.Scopes) == 0 {
		return nil // no restrictions
	}
	allowed := make(map[string]bool)
	for _, s := range client.Scopes {
		allowed[s] = true
	}
	for _, s := range requested {
		if !allowed[s] {
			return errInvalidScope
		}
	}
	return nil
}

// RevokeToken handles DELETE /oauth/tokens/{id}.
func (m *Manager) RevokeToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m.mu.Lock()
	if tok, ok := m.tokens[id]; ok {
		tok.Revoked = true
	}
	m.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// RevokeRFC7009 handles POST /oauth/revoke per RFC 7009.
// Accepts token via form body, authenticates client, revokes access or refresh token.
// Always returns 200 per spec (even if token doesn't exist — prevents token scanning).
func (m *Manager) RevokeRFC7009(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tokenValue := r.FormValue("token")
	tokenType := r.FormValue("token_type_hint") // "access_token" or "refresh_token"
	clientID, clientSecret, hasBasic := r.BasicAuth()
	if !hasBasic {
		clientID = r.FormValue("client_id")
		clientSecret = r.FormValue("client_secret")
	}

	if clientID != "" {
		if err := m.AuthenticateClient(clientID, clientSecret); err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_client", "authentication failed")
			return
		}
	}

	if tokenValue == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	m.mu.Lock()
	// Try as access token
	if tok, ok := m.tokens[tokenValue]; ok {
		tok.Revoked = true
	}
	// Try as refresh token
	if tokenType == "refresh_token" || tokenType == "" {
		for _, tok := range m.tokens {
			if tok.RefreshToken == tokenValue {
				tok.Revoked = true
			}
		}
	}
	m.mu.Unlock()

	// RFC 7009 §2.1: always 200, even if token invalid
	w.WriteHeader(http.StatusOK)
}

// ListTokens handles GET /oauth/tokens.
func (m *Manager) ListTokens(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	var list []Token
	for _, t := range m.tokens {
		if !t.Revoked {
			list = append(list, *t) // copy to avoid race
		}
	}
	m.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// GetToken handles GET /oauth/tokens/{id}.
func (m *Manager) GetToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m.mu.RLock()
	tok, ok := m.tokens[id]
	var copy Token
	if ok {
		copy = *tok
	}
	m.mu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "token not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(copy)
}

var (
	errInvalidClient = &oauthError{Code: "invalid_client"}
	errInvalidScope  = &oauthError{Code: "invalid_scope"}
)

// Lookup returns token metadata for introspection.
func (m *Manager) Lookup(id string) (clientID string, scopes []string, createdAt, expiresAt time.Time, revoked bool, ok bool) {
	m.mu.RLock()
	tok, exists := m.tokens[id]
	m.mu.RUnlock()
	if !exists {
		return "", nil, time.Time{}, time.Time{}, false, false
	}
	return tok.ClientID, tok.Scopes, tok.CreatedAt, tok.ExpiresAt, tok.Revoked, true
}

type oauthError struct{ Code string }

func (e *oauthError) Error() string { return e.Code }

func writeError(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": desc,
	})
}

func generateID(prefix string) string {
	b := make([]byte, 16)
	rand.Read(b)
	return prefix + hex.EncodeToString(b)
}
