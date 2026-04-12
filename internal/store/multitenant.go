package store

import "sync"

// MultiTenant wraps per-tenant stores with automatic creation.
type MultiTenant struct {
	mu      sync.RWMutex
	tenants map[string]Store
	factory func(tenant string) Store
}

// NewMultiTenant creates a multi-tenant store using factory for new tenants.
func NewMultiTenant(factory func(tenant string) Store) *MultiTenant {
	return &MultiTenant{
		tenants: make(map[string]Store),
		factory: factory,
	}
}

// For returns the store for a tenant, creating it if needed.
func (m *MultiTenant) For(tenant string) Store {
	m.mu.RLock()
	s, ok := m.tenants[tenant]
	m.mu.RUnlock()
	if ok {
		return s
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok = m.tenants[tenant]; ok {
		return s
	}
	s = m.factory(tenant)
	m.tenants[tenant] = s
	return s
}

// Tenants returns the list of active tenant IDs.
func (m *MultiTenant) Tenants() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.tenants))
	for k := range m.tenants {
		ids = append(ids, k)
	}
	return ids
}

// CloseAll closes all tenant stores.
func (m *MultiTenant) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var last error
	for _, s := range m.tenants {
		if err := s.Close(); err != nil {
			last = err
		}
	}
	return last
}
