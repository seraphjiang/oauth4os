// Package keyring manages RSA signing keys with scheduled rotation
// and serves the public keys via /.well-known/jwks.json.
package keyring

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// Key is a timestamped RSA key pair.
type Key struct {
	ID        string
	Private   *rsa.PrivateKey
	CreatedAt time.Time
}

// Ring holds the active and previous signing keys.
type Ring struct {
	mu       sync.RWMutex
	current  *Key
	previous *Key
	bits     int
	interval time.Duration
	stopCh   chan struct{}
}

// New creates a Ring, generates the first key, and starts rotation.
func New(bits int, rotateEvery time.Duration) (*Ring, error) {
	r := &Ring{bits: bits, interval: rotateEvery, stopCh: make(chan struct{})}
	if err := r.rotate(); err != nil {
		return nil, err
	}
	go r.rotateLoop()
	return r, nil
}

// Current returns the active signing key.
func (r *Ring) Current() *Key {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.current
}

// Stop halts the rotation goroutine.
func (r *Ring) Stop() {
	close(r.stopCh)
}

func (r *Ring) rotate() error {
	priv, err := rsa.GenerateKey(rand.Reader, r.bits)
	if err != nil {
		return err
	}
	kid := genKID(&priv.PublicKey)
	k := &Key{ID: kid, Private: priv, CreatedAt: time.Now()}
	r.mu.Lock()
	r.previous = r.current
	r.current = k
	r.mu.Unlock()
	return nil
}

func (r *Ring) rotateLoop() {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.rotate()
		case <-r.stopCh:
			return
		}
	}
}

// JWKSHandler returns an http.HandlerFunc serving /.well-known/jwks.json.
func (r *Ring) JWKSHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		r.mu.RLock()
		var keys []jwk
		if r.current != nil {
			keys = append(keys, toJWK(r.current))
		}
		if r.previous != nil {
			keys = append(keys, toJWK(r.previous))
		}
		r.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		json.NewEncoder(w).Encode(jwks{Keys: keys})
	}
}

// jwk is a JSON Web Key (RSA public key).
type jwk struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

func toJWK(k *Key) jwk {
	pub := &k.Private.PublicKey
	return jwk{
		Kty: "RSA",
		Use: "sig",
		Kid: k.ID,
		Alg: "RS256",
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}

func genKID(pub *rsa.PublicKey) string {
	h := sha256.Sum256(pub.N.Bytes())
	return base64.RawURLEncoding.EncodeToString(h[:8])
}
