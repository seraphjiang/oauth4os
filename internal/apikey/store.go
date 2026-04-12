// Package apikey provides X-API-Key header authentication for machine-to-machine access.
package apikey

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// Key represents a stored API key.
type Key struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	Prefix    string    `json:"prefix"` // first 8 chars for display
	Hash      string    `json:"-"`      // full key, constant-time compared
	Scopes    []string  `json:"scopes"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used,omitempty"`
	Revoked   bool      `json:"revoked"`
}

// Claims returned on successful API key auth.
type Claims struct {
	ClientID string
	Scopes   []string
	KeyID    string
}

// Store manages API keys.
type Store struct {
	mu   sync.RWMutex
	keys map[string]*Key // full key → Key
}

// NewStore creates an API key store.
func NewStore() *Store {
	return &Store{keys: make(map[string]*Key)}
}

// Generate creates a new API key for a client. Returns the raw key (show once).
func (s *Store) Generate(clientID string, scopes []string) (rawKey string, k *Key) {
	b := make([]byte, 32)
	rand.Read(b)
	raw := "oak_" + hex.EncodeToString(b) // oak = oauth4os api key
	k = &Key{
		ID:        hex.EncodeToString(b[:8]),
		ClientID:  clientID,
		Prefix:    raw[:12],
		Hash:      raw,
		Scopes:    scopes,
		CreatedAt: time.Now(),
	}
	s.mu.Lock()
	s.keys[raw] = k
	s.mu.Unlock()
	return raw, k
}

// Validate checks an API key. Returns claims if valid.
func (s *Store) Validate(rawKey string) (*Claims, bool) {
	s.mu.RLock()
	k, ok := s.keys[rawKey]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if k.Revoked {
		return nil, false
	}
	if subtle.ConstantTimeCompare([]byte(rawKey), []byte(k.Hash)) != 1 {
		return nil, false
	}
	s.mu.Lock()
	k.LastUsed = time.Now()
	s.mu.Unlock()
	return &Claims{ClientID: k.ClientID, Scopes: k.Scopes, KeyID: k.ID}, true
}

// Revoke invalidates an API key by its ID.
func (s *Store) Revoke(keyID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range s.keys {
		if k.ID == keyID {
			k.Revoked = true
			return true
		}
	}
	return false
}

// List returns all keys for a client (without hashes).
func (s *Store) List(clientID string) []*Key {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Key
	for _, k := range s.keys {
		if k.ClientID == clientID && !k.Revoked {
			out = append(out, k)
		}
	}
	return out
}

// ExtractKey gets the API key from the request (X-API-Key header).
func ExtractKey(r *http.Request) string {
	return r.Header.Get("X-API-Key")
}
