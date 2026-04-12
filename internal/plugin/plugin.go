// Package plugin provides a runtime plugin system for custom auth logic.
// Plugins implement the Authorizer interface and are loaded from shared objects.
package plugin

import (
	"fmt"
	"net/http"
	"plugin"
	"sync"
)

// Authorizer is the interface plugins must implement.
type Authorizer interface {
	// Authorize returns nil to allow, non-nil error to deny.
	Authorize(r *http.Request, claims map[string]interface{}) error
	Name() string
}

// Registry holds loaded plugins.
type Registry struct {
	mu      sync.RWMutex
	plugins []Authorizer
}

func NewRegistry() *Registry {
	return &Registry{}
}

// Load opens a .so file and looks up the "NewAuthorizer" symbol.
func (reg *Registry) Load(path string) error {
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("plugin open %s: %w", path, err)
	}
	sym, err := p.Lookup("NewAuthorizer")
	if err != nil {
		return fmt.Errorf("plugin %s: missing NewAuthorizer: %w", path, err)
	}
	fn, ok := sym.(func() Authorizer)
	if !ok {
		return fmt.Errorf("plugin %s: NewAuthorizer has wrong signature", path)
	}
	auth := fn()
	reg.mu.Lock()
	reg.plugins = append(reg.plugins, auth)
	reg.mu.Unlock()
	return nil
}

// Register adds a built-in authorizer (no .so needed).
func (reg *Registry) Register(a Authorizer) {
	reg.mu.Lock()
	reg.plugins = append(reg.plugins, a)
	reg.mu.Unlock()
}

// Authorize runs all plugins. First denial wins.
func (reg *Registry) Authorize(r *http.Request, claims map[string]interface{}) error {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	for _, p := range reg.plugins {
		if err := p.Authorize(r, claims); err != nil {
			return fmt.Errorf("plugin %s: %w", p.Name(), err)
		}
	}
	return nil
}

// List returns names of loaded plugins.
func (reg *Registry) List() []string {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	names := make([]string, len(reg.plugins))
	for i, p := range reg.plugins {
		names[i] = p.Name()
	}
	return names
}
