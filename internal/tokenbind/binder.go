// Package tokenbind implements token binding — binds tokens to a client fingerprint
// (IP + User-Agent hash) to prevent stolen token reuse from different clients.
package tokenbind

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
)

// Binder tracks token-to-fingerprint bindings.
type Binder struct {
	mu       sync.RWMutex
	bindings map[string]string // token prefix → fingerprint hash
}

// New creates a token binder.
func New() *Binder {
	return &Binder{bindings: make(map[string]string)}
}

// Fingerprint computes a client fingerprint from the request.
func Fingerprint(r *http.Request) string {
	h := sha256.New()
	h.Write([]byte(r.RemoteAddr))
	h.Write([]byte(r.UserAgent()))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// Bind associates a token with a fingerprint on first use only.
func (b *Binder) Bind(tokenPrefix, fingerprint string) {
	b.mu.Lock()
	if _, exists := b.bindings[tokenPrefix]; !exists {
		b.bindings[tokenPrefix] = fingerprint
	}
	b.mu.Unlock()
}

// Verify checks if a token matches its bound fingerprint.
// Returns true if no binding exists (unbound tokens are allowed) or if fingerprint matches.
func (b *Binder) Verify(tokenPrefix, fingerprint string) bool {
	b.mu.RLock()
	bound, ok := b.bindings[tokenPrefix]
	b.mu.RUnlock()
	if !ok {
		return true // no binding = allow
	}
	return bound == fingerprint
}

// Remove deletes a binding (on token revocation).
func (b *Binder) Remove(tokenPrefix string) {
	b.mu.Lock()
	delete(b.bindings, tokenPrefix)
	b.mu.Unlock()
}
