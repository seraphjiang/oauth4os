package scope

import (
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

func TestMapGlobal(t *testing.T) {
	m := NewMapper(map[string]config.Role{
		"read:logs-*": {BackendRoles: []string{"logs_read"}},
		"admin":       {BackendRoles: []string{"all_access"}},
	})
	roles := m.Map([]string{"read:logs-*"})
	if len(roles) != 1 || roles[0] != "logs_read" {
		t.Fatalf("expected [logs_read], got %v", roles)
	}
}

func TestMapForIssuerTenant(t *testing.T) {
	m := NewMultiTenantMapper(
		map[string]config.Role{"admin": {BackendRoles: []string{"global_admin"}}},
		map[string]config.Tenant{
			"https://keycloak.example.com": {
				ScopeMapping: map[string]config.Role{
					"admin": {BackendRoles: []string{"tenant_admin"}},
				},
			},
		},
	)
	// Tenant issuer → tenant mapping
	roles := m.MapForIssuer("https://keycloak.example.com", []string{"admin"})
	if len(roles) != 1 || roles[0] != "tenant_admin" {
		t.Fatalf("expected [tenant_admin], got %v", roles)
	}
	// Unknown issuer → global fallback
	roles = m.MapForIssuer("https://other.example.com", []string{"admin"})
	if len(roles) != 1 || roles[0] != "global_admin" {
		t.Fatalf("expected [global_admin], got %v", roles)
	}
}

func TestMapForIssuerDedup(t *testing.T) {
	m := NewMapper(map[string]config.Role{
		"read:a": {BackendRoles: []string{"shared"}},
		"read:b": {BackendRoles: []string{"shared"}},
	})
	roles := m.Map([]string{"read:a", "read:b"})
	if len(roles) != 1 {
		t.Fatalf("expected deduped [shared], got %v", roles)
	}
}
