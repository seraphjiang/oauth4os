package token

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ClientStore persists registered clients to a JSON file.
// Writes are atomic (temp + rename). A .bak backup is kept.
type ClientStore struct {
	path string
	mu   sync.Mutex
}

// NewClientStore creates a store backed by the given file path.
// If the file is corrupt, it attempts recovery from .bak.
func NewClientStore(path string, mgr *Manager) (*ClientStore, error) {
	s := &ClientStore{path: path}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	clients, err := s.load(path)
	if err != nil {
		// Try backup
		clients, err = s.load(path + ".bak")
		if err != nil {
			// No valid data — start fresh
			return s, nil
		}
	}
	for _, c := range clients {
		mgr.RegisterClient(c.ID, c.Secret, c.Scopes, c.RedirectURIs)
	}
	return s, nil
}

func (s *ClientStore) load(path string) ([]*Client, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty file")
	}
	var clients []*Client
	if err := json.Unmarshal(data, &clients); err != nil {
		return nil, err
	}
	return clients, nil
}

// Save writes the current client list to disk atomically.
// Creates a .bak of the previous version before overwriting.
func (s *ClientStore) Save(mgr *Manager) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(mgr.Clients(), "", "  ")
	if err != nil {
		return err
	}

	// Backup existing file (best-effort)
	if _, err := os.Stat(s.path); err == nil {
		os.Rename(s.path, s.path+".bak")
	}

	// Atomic write: temp file + rename
	tmp := s.path + ".tmp." + fmt.Sprintf("%d", time.Now().UnixNano())
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		os.Remove(tmp) // clean up orphaned temp file
		return err
	}
	return nil
}
