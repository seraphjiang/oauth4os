package token

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// ClientStore persists registered clients to a JSON file.
type ClientStore struct {
	path string
	mu   sync.Mutex
}

// NewClientStore creates a store backed by the given file path.
// If the file exists, clients are loaded into the manager.
func NewClientStore(path string, mgr *Manager) (*ClientStore, error) {
	s := &ClientStore{path: path}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		var clients []*Client
		if err := json.Unmarshal(data, &clients); err != nil {
			return nil, err
		}
		for _, c := range clients {
			mgr.RegisterClient(c.ID, c.Secret, c.Scopes, c.RedirectURIs)
		}
	}
	return s, nil
}

// Save writes the current client list to disk.
func (s *ClientStore) Save(mgr *Manager) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	clients := mgr.Clients()
	data, err := json.MarshalIndent(clients, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
