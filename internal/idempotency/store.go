// Package idempotency provides request deduplication via Idempotency-Key headers.
package idempotency

import (
	"net/http"
	"sync"
	"time"
)

// entry stores a cached response for a given idempotency key.
type entry struct {
	status  int
	headers map[string]string
	body    []byte
	expires time.Time
}

// Store tracks idempotency keys and their responses.
type Store struct {
	mu      sync.RWMutex
	entries map[string]*entry
	ttl     time.Duration
	stopCh  chan struct{}
}

// New creates a store with the given TTL for idempotency keys.
func New(ttl time.Duration) *Store {
	s := &Store{entries: make(map[string]*entry), ttl: ttl, stopCh: make(chan struct{})}
	go s.reap()
	return s
}

// Stop halts the background reaper goroutine.
func (s *Store) Stop() { close(s.stopCh) }

// Middleware returns HTTP middleware that deduplicates requests with Idempotency-Key header.
// Only applies to POST/PUT/PATCH methods.
func (s *Store) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" && r.Method != "PUT" && r.Method != "PATCH" {
			next.ServeHTTP(w, r)
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check for cached response
		s.mu.RLock()
		e, ok := s.entries[key]
		s.mu.RUnlock()
		if ok && time.Now().Before(e.expires) {
			for k, v := range e.headers {
				w.Header().Set(k, v)
			}
			w.Header().Set("X-Idempotent-Replay", "true")
			w.WriteHeader(e.status)
			w.Write(e.body)
			return
		}

		// Capture response
		rec := &recorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)

		// Store response
		s.mu.Lock()
		s.entries[key] = &entry{
			status:  rec.status,
			body:    rec.body,
			headers: map[string]string{"Content-Type": w.Header().Get("Content-Type")},
			expires: time.Now().Add(s.ttl),
		}
		s.mu.Unlock()
	})
}

func (s *Store) reap() {
	if s.ttl <= 0 {
		<-s.stopCh
		return
	}
	ticker := time.NewTicker(s.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			s.mu.Lock()
			for k, v := range s.entries {
				if now.After(v.expires) {
					delete(s.entries, k)
				}
			}
			s.mu.Unlock()
		case <-s.stopCh:
			return
		}
	}
}

type recorder struct {
	http.ResponseWriter
	status int
	body   []byte
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return r.ResponseWriter.Write(b)
}
