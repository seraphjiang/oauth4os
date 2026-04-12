// Package introspect implements RFC 7662 Token Introspection.
package introspect

import (
	"encoding/json"
	"net/http"
	"strings"
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
type Handler struct {
	lookup TokenLookup
}

// NewHandler creates an introspection handler.
func NewHandler(lookup TokenLookup) *Handler {
	return &Handler{lookup: lookup}
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
