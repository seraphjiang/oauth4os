package scope

import (
	"sync"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

func TestMapEmptyScopes(t *testing.T) {
	m := NewMapper(map[string]config.Role{
		"admin": {BackendUser: "admin", BackendRoles: []string{"all_access"}},
	})
	roles := m.Map(nil)
	if len(roles) != 0 {
		t.Fatalf("empty scopes should return empty roles, got %v", roles)
	}
}

func TestMapUnknownScope(t *testing.T) {
	m := NewMapper(map[string]config.Role{
		"admin": {BackendUser: "admin", BackendRoles: []string{"all_access"}},
	})
	roles := m.Map([]string{"unknown_scope"})
	if len(roles) != 0 {
		t.Fatalf("unknown scope should return empty roles, got %v", roles)
	}
}

func TestMapMultipleScopes(t *testing.T) {
	m := NewMapper(map[string]config.Role{
		"read:logs-*": {BackendRoles: []string{"readall"}},
		"admin":       {BackendRoles: []string{"all_access"}},
	})
	roles := m.Map([]string{"read:logs-*", "admin"})
	if len(roles) < 2 {
		t.Fatalf("expected at least 2 roles, got %v", roles)
	}
}

func TestConcurrentMap(t *testing.T) {
	m := NewMapper(map[string]config.Role{
		"read:logs-*": {BackendUser: "reader", BackendRoles: []string{"readall"}},
	})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			roles := m.Map([]string{"read:logs-*"})
			if len(roles) == 0 {
				t.Error("expected roles")
			}
		}()
	}
	wg.Wait()
}

func TestMapForUnknownIssuer(t *testing.T) {
	m := NewMultiTenantMapper(
		map[string]config.Role{"admin": {BackendRoles: []string{"all_access"}}},
		map[string]config.Tenant{"https://known.com": {Scopes: map[string]config.Role{"admin": {BackendRoles: []string{"tenant_admin"}}}}},
	)
	// Unknown issuer falls back to global
	roles := m.MapForIssuer("https://unknown.com", []string{"admin"})
	found := false
	for _, r := range roles {
		if r == "all_access" {
			found = true
		}
	}
	if !found {
		t.Fatalf("unknown issuer should fall back to global mapping, got %v", roles)
	}
}
