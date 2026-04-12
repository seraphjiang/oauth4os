// Package register implements RFC 7591 Dynamic Client Registration.
package register

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

// ClientStore is the interface for persisting clients.
type ClientStore interface {
	RegisterClient(id, secret string, scopes, redirectURIs []string)
}

// Request is the RFC 7591 registration request.
type Request struct {
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scope        string   `json:"scope"`
	GrantTypes   []string `json:"grant_types"`
	TokenAuth    string   `json:"token_endpoint_auth_method"`
}

// Response is the RFC 7591 registration response.
type Response struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris,omitempty"`
	Scope        string   `json:"scope"`
	GrantTypes   []string `json:"grant_types"`
}

// Handler handles POST /oauth/register.
type Handler struct {
	store         ClientStore
	allowedScopes map[string]bool // scope allowlist; empty = allow all
}

// NewHandler creates a registration handler with optional scope allowlist.
func NewHandler(store ClientStore, allowedScopes []string) *Handler {
	m := make(map[string]bool)
	for _, s := range allowedScopes {
		m[s] = true
	}
	return &Handler{store: store, allowedScopes: m}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "invalid_client_metadata", "malformed JSON")
		return
	}

	// Validate redirect URIs
	for _, uri := range req.RedirectURIs {
		if uri == "" || strings.Contains(uri, "..") {
			writeErr(w, 400, "invalid_redirect_uri", "invalid redirect_uri")
			return
		}
	}

	// Validate scopes against allowlist
	scopes := strings.Fields(req.Scope)
	if len(h.allowedScopes) > 0 {
		for _, s := range scopes {
			if !h.allowedScopes[s] {
				writeErr(w, 400, "invalid_client_metadata", "scope not allowed: "+s)
				return
			}
		}
	}

	// Default grant types
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"client_credentials"}
	}

	clientID := "client_" + randomHex(8)
	clientSecret := randomHex(32)

	h.store.RegisterClient(clientID, clientSecret, scopes, req.RedirectURIs)

	resp := Response{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		ClientName:   req.ClientName,
		RedirectURIs: req.RedirectURIs,
		Scope:        strings.Join(scopes, " "),
		GrantTypes:   req.GrantTypes,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func writeErr(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": code, "error_description": desc})
}
