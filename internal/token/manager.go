package token

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

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
	ID     string
	Secret string
	Scopes []string // allowed scopes
}

// Manager handles token lifecycle.
type Manager struct {
	tokens   map[string]*Token
	refresh  map[string]string // refresh_token -> token_id
	clients  map[string]*Client
	mu       sync.RWMutex
}

// NewManager creates a token manager.
func NewManager() *Manager {
	return &Manager{
		tokens:  make(map[string]*Token),
		refresh: make(map[string]string),
		clients: make(map[string]*Client),
	}
}

// RegisterClient adds a client for authentication.
func (m *Manager) RegisterClient(id, secret string, scopes []string) {
	m.mu.Lock()
	m.clients[id] = &Client{ID: id, Secret: secret, Scopes: scopes}
	m.mu.Unlock()
}

// IsValid checks if a token ID is valid (not expired, not revoked).
func (m *Manager) IsValid(tokenID string) bool {
	m.mu.RLock()
	tok, ok := m.tokens[tokenID]
	m.mu.RUnlock()
	return ok && !tok.Revoked && time.Now().Before(tok.ExpiresAt)
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

	// Authenticate client
	if err := m.authenticateClient(clientID, clientSecret); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_client", "authentication failed")
		return
	}

	// Validate requested scopes
	scopes := strings.Fields(scopeStr)
	if err := m.validateScopes(clientID, scopes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_scope", "requested scope exceeds client allowance")
		return
	}

	tok, refreshTok := m.createToken(clientID, scopes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  tok.ID,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": refreshTok,
		"scope":         scopeStr,
	})
}

func (m *Manager) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.FormValue("refresh_token")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	if err := m.authenticateClient(clientID, clientSecret); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_client", "authentication failed")
		return
	}

	m.mu.Lock()
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
	m.mu.Unlock()

	tok, newRefresh := m.createToken(clientID, oldToken.Scopes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  tok.ID,
		"token_type":    "Bearer",
		"expires_in":    3600,
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
	m.mu.Unlock()

	return tok, refreshTok
}

func (m *Manager) authenticateClient(clientID, secret string) error {
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

// ListTokens handles GET /oauth/tokens.
func (m *Manager) ListTokens(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	var list []*Token
	for _, t := range m.tokens {
		if !t.Revoked {
			list = append(list, t)
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
	m.mu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "token not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tok)
}

var (
	errInvalidClient = &oauthError{Code: "invalid_client"}
	errInvalidScope  = &oauthError{Code: "invalid_scope"}
)

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
