// Package session tracks active client sessions with limits and force logout.
package session

import (
	"sync"
	"time"
)

// Session represents an active client session.
type Session struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	TokenID   string    `json:"token_id"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
}

// Manager tracks active sessions per client.
type Manager struct {
	sessions map[string]*Session // session_id → session
	mu       sync.RWMutex
	limits   map[string]int // client_id → max concurrent sessions; "*" for global default
}

// New creates a session manager. limits maps client_id (or "*") to max sessions.
func New(limits map[string]int) *Manager {
	if limits == nil {
		limits = map[string]int{"*": 100}
	}
	return &Manager{sessions: make(map[string]*Session), limits: limits}
}

// Create registers a new session. Returns false if client is at session limit.
func (m *Manager) Create(sessionID, clientID, tokenID, ip string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	limit := m.getLimit(clientID)
	if limit > 0 && m.countLocked(clientID) >= limit {
		return false
	}

	m.sessions[sessionID] = &Session{
		ID:        sessionID,
		ClientID:  clientID,
		TokenID:   tokenID,
		IP:        ip,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}
	return true
}

// Touch updates last_seen for a session.
func (m *Manager) Touch(sessionID string) {
	m.mu.Lock()
	if s, ok := m.sessions[sessionID]; ok {
		s.LastSeen = time.Now()
	}
	m.mu.Unlock()
}

// Remove deletes a single session.
func (m *Manager) Remove(sessionID string) {
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
}

// ForceLogout removes all sessions for a client.
func (m *Manager) ForceLogout(clientID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for id, s := range m.sessions {
		if s.ClientID == clientID {
			delete(m.sessions, id)
			count++
		}
	}
	return count
}

// List returns all active sessions for a client. Empty clientID returns all.
func (m *Manager) List(clientID string) []Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Session
	for _, s := range m.sessions {
		if clientID == "" || s.ClientID == clientID {
			result = append(result, *s)
		}
	}
	return result
}

// Count returns active session count for a client.
func (m *Manager) Count(clientID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.countLocked(clientID)
}

// Cleanup removes sessions idle longer than maxIdle.
func (m *Manager) Cleanup(maxIdle time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxIdle)
	count := 0
	for id, s := range m.sessions {
		if s.LastSeen.Before(cutoff) {
			delete(m.sessions, id)
			count++
		}
	}
	return count
}

func (m *Manager) countLocked(clientID string) int {
	n := 0
	for _, s := range m.sessions {
		if s.ClientID == clientID {
			n++
		}
	}
	return n
}

func (m *Manager) getLimit(clientID string) int {
	if l, ok := m.limits[clientID]; ok {
		return l
	}
	if l, ok := m.limits["*"]; ok {
		return l
	}
	return 0 // no limit
}
