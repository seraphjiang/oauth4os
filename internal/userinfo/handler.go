// Package userinfo implements the OIDC UserInfo endpoint (§5.3).
package userinfo

import (
	"encoding/json"
	"net/http"
	"strings"
)

// TokenLookup resolves a Bearer token to claims.
type TokenLookup func(token string) (clientID string, scopes []string, ok bool)

// Handler serves GET/POST /oauth/userinfo per OIDC Core §5.3.
type Handler struct {
	lookup TokenLookup
}

func New(lookup TokenLookup) *Handler { return &Handler{lookup: lookup} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := ""
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		token = strings.TrimPrefix(auth, "Bearer ")
	}
	if token == "" {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		http.Error(w, `{"error":"invalid_token"}`, 401)
		return
	}

	clientID, scopes, ok := h.lookup(token)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		http.Error(w, `{"error":"invalid_token"}`, 401)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sub":   clientID,
		"scope": strings.Join(scopes, " "),
	})
}
