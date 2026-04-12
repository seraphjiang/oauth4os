// Package introspect implements RFC 7662 Token Introspection.
package introspect

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TokenLookup resolves a token string to its metadata.
type TokenLookup interface {
	Introspect(token string) *Response
}

// Response is the RFC 7662 introspection response.
type Response struct {
	Active    bool     `json:"active"`
	Scope     string   `json:"scope,omitempty"`
	ClientID  string   `json:"client_id,omitempty"`
	Sub       string   `json:"sub,omitempty"`
	Exp       int64    `json:"exp,omitempty"`
	Iat       int64    `json:"iat,omitempty"`
	Iss       string   `json:"iss,omitempty"`
	TokenType string   `json:"token_type,omitempty"`
}

// Handler handles POST /oauth/introspect.
// ClientAuthenticator validates client credentials.
type ClientAuthenticator func(clientID, clientSecret string) error

type Handler struct {
	lookup     TokenLookup
	clientAuth ClientAuthenticator // nil = no auth required (backward compatible)
}

// NewHandler creates an introspection handler.
func NewHandler(lookup TokenLookup) *Handler {
	return &Handler{lookup: lookup}
}

// SetClientAuth enables client authentication on the introspection endpoint (RFC 7662 §2.1).
func (h *Handler) SetClientAuth(auth ClientAuthenticator) {
	h.clientAuth = auth
}

// ServeHTTP implements RFC 7662.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	tokenStr := r.FormValue("token")
	if tokenStr == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Active: false})
		return
	}

	resp := h.lookup.Introspect(tokenStr)
	if resp == nil {
		resp = &Response{Active: false}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ManagerAdapter adapts token.Manager to TokenLookup.
type ManagerAdapter struct {
	GetToken func(id string) (clientID string, scopes []string, createdAt, expiresAt time.Time, revoked bool, ok bool)
}

func (a *ManagerAdapter) Introspect(tokenStr string) *Response {
	clientID, scopes, createdAt, expiresAt, revoked, ok := a.GetToken(tokenStr)
	if !ok || revoked || time.Now().After(expiresAt) {
		return &Response{Active: false}
	}
	return &Response{
		Active:    true,
		Scope:     strings.Join(scopes, " "),
		ClientID:  clientID,
		Sub:       clientID,
		Exp:       expiresAt.Unix(),
		Iat:       createdAt.Unix(),
		TokenType: "Bearer",
	}
}

// CachedLookup wraps a TokenLookup with a TTL cache.
type CachedLookup struct {
	inner TokenLookup
	ttl   time.Duration
	mu    sync.RWMutex
	cache map[string]*cacheEntry
}

type cacheEntry struct {
	resp *Response
	at   time.Time
}

// NewCachedLookup wraps a lookup with caching. TTL of 0 defaults to 30s.
func NewCachedLookup(inner TokenLookup, ttl time.Duration) *CachedLookup {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &CachedLookup{inner: inner, ttl: ttl, cache: make(map[string]*cacheEntry)}
}

func (c *CachedLookup) Introspect(token string) *Response {
	c.mu.RLock()
	if e, ok := c.cache[token]; ok && time.Since(e.at) < c.ttl {
		c.mu.RUnlock()
		return e.resp
	}
	c.mu.RUnlock()

	resp := c.inner.Introspect(token)

	c.mu.Lock()
	c.cache[token] = &cacheEntry{resp: resp, at: time.Now()}
	c.mu.Unlock()
	return resp
}
