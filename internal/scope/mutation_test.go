package scope

import (
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

var testMapping = map[string]config.Role{
	"read:logs-*":  {BackendRoles: []string{"readall"}},
	"write:logs-*": {BackendRoles: []string{"writeall"}},
	"admin":        {BackendRoles: []string{"readall", "writeall", "admin"}},
}

// Mutation: remove dedup → duplicate roles must be deduplicated
func TestMutation_Dedup(t *testing.T) {
	m := NewMapper(testMapping)
	roles := m.Map([]string{"read:logs-*", "admin"}) // both map to readall
	count := 0
	for _, r := range roles {
		if r == "readall" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("readall should appear once, got %d", count)
	}
}

// Mutation: remove tenant lookup → MapForIssuer must use tenant mapping
func TestMutation_TenantMapping(t *testing.T) {
	tenants := map[string]config.Tenant{
		"https://idp.example.com": {ScopeMapping: map[string]config.Role{
			"read": {BackendRoles: []string{"tenant-read"}},
		}},
	}
	m := NewMultiTenantMapper(testMapping, tenants)
	roles := m.MapForIssuer("https://idp.example.com", []string{"read"})
	if len(roles) != 1 || roles[0] != "tenant-read" {
		t.Errorf("expected tenant-read, got %v", roles)
	}
}

// Mutation: remove global fallback → unknown issuer must fall back to global
func TestMutation_GlobalFallback(t *testing.T) {
	m := NewMultiTenantMapper(testMapping, nil)
	roles := m.MapForIssuer("https://unknown.com", []string{"read:logs-*"})
	if len(roles) != 1 || roles[0] != "readall" {
		t.Errorf("expected global fallback to readall, got %v", roles)
	}
}

// Mutation: remove mapping lookup → unknown scope must return empty
func TestMutation_UnknownScope(t *testing.T) {
	m := NewMapper(testMapping)
	roles := m.Map([]string{"nonexistent"})
	if len(roles) != 0 {
		t.Errorf("unknown scope should return empty, got %v", roles)
	}
}

// Mutation: remove Map expansion → must expand scope to backend roles
func TestMutation_MapExpansion(t *testing.T) {
	m := NewMapper(map[string]config.Role{
		"admin": {BackendUser: "admin", BackendRoles: []string{"all_access"}},
	})
	result := m.Map([]string{"admin"})
	if len(result) == 0 {
		t.Error("admin scope should map to backend roles")
	}
}
