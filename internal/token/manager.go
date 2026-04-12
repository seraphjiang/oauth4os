package token

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type Token struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	Scopes    []string  `json:"scopes"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
}

type Manager struct {
	tokens map[string]*Token
	mu     sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{tokens: make(map[string]*Token)}
}

func (m *Manager) IssueToken(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	clientID := r.FormValue("client_id")
	scopeStr := r.FormValue("scope")

	id := generateID()
	tok := &Token{
		ID:        id,
		ClientID:  clientID,
		Scopes:    splitScopes(scopeStr),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	m.mu.Lock()
	m.tokens[id] = tok
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": id,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        scopeStr,
	})
}

func (m *Manager) RevokeToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m.mu.Lock()
	if tok, ok := m.tokens[id]; ok {
		tok.Revoked = true
	}
	m.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

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

func (m *Manager) GetToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m.mu.RLock()
	tok, ok := m.tokens[id]
	m.mu.RUnlock()
	if !ok {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tok)
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "tok_" + hex.EncodeToString(b)
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
