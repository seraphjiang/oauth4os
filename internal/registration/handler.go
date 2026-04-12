// Package registration implements RFC 7591 OAuth 2.0 Dynamic Client Registration.
// Clients self-register via POST /oauth/register and receive client_id + client_secret.
package registration

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Request is the RFC 7591 client registration request.
type Request struct {
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris,omitempty"`
	GrantTypes   []string `json:"grant_types,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	TokenEPAuth  string   `json:"token_endpoint_auth_method,omitempty"`
}

// Response is the RFC 7591 client registration response.
type Response struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret,omitempty"`
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris,omitempty"`
	GrantTypes   []string `json:"grant_types"`
	Scope        string   `json:"scope,omitempty"`
	IssuedAt     int64    `json:"client_id_issued_at"`
	SecretExpiry int64    `json:"client_secret_expires_at"` // 0 = never
}

// ClientRegistrar registers clients and notifies the token manager.
type ClientRegistrar func(id, secret string, scopes, redirectURIs []string)

// Handler handles dynamic client registration.
type Handler struct {
	mu            sync.RWMutex
	clients       map[string]*Response
	register      ClientRegistrar
	defaults      []string // default grant types
	allowedScopes map[string]bool
}

// NewHandler creates a registration handler. allowedScopes restricts which scopes clients can request (nil = allow all).
func NewHandler(register ClientRegistrar, allowedScopes []string) *Handler {
	m := make(map[string]bool)
	for _, s := range allowedScopes {
		m[s] = true
	}
	return &Handler{
		clients:       make(map[string]*Response),
		register:      register,
		defaults:      []string{"client_credentials"},
		allowedScopes: m,
	}
}

// Register handles POST /oauth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "invalid_client_metadata", "malformed JSON")
		return
	}
	if req.ClientName == "" {
		writeErr(w, 400, "invalid_client_metadata", "client_name required")
		return
	}

	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = h.defaults
	}

	clientID := "client_" + randomHex(8)
	clientSecret := randomHex(32)

	resp := &Response{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		ClientName:   req.ClientName,
		RedirectURIs: req.RedirectURIs,
		GrantTypes:   grantTypes,
		Scope:        req.Scope,
		IssuedAt:     time.Now().Unix(),
		SecretExpiry: 0,
	}

	// Parse scopes for token manager
	var scopes []string
	if req.Scope != "" {
		for _, s := range splitScope(req.Scope) {
			if len(h.allowedScopes) > 0 && !h.allowedScopes[s] {
				writeErr(w, 400, "invalid_client_metadata", "scope not allowed: "+s)
				return
			}
			scopes = append(scopes, s)
		}
	}

	h.mu.Lock()
	h.clients[clientID] = resp
	h.mu.Unlock()

	// Register with token manager so client can authenticate
	h.register(clientID, clientSecret, scopes, req.RedirectURIs)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// Get handles GET /oauth/register/{client_id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	clientID := r.PathValue("client_id")
	h.mu.RLock()
	client, ok := h.clients[clientID]
	h.mu.RUnlock()
	if !ok {
		writeErr(w, 404, "invalid_client", "client not found")
		return
	}
	// Don't return secret on GET
	safe := *client
	safe.ClientSecret = ""
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(safe)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func splitScope(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == ' ' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func writeErr(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": code, "error_description": desc})
}
