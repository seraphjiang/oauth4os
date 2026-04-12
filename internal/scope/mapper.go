package scope

import "github.com/seraphjiang/oauth4os/internal/config"

// Mapper resolves scopes to OpenSearch backend roles.
// Supports per-tenant (per-issuer) scope mappings with global fallback.
type Mapper struct {
	global  map[string]config.Role
	tenants map[string]map[string]config.Role // issuer → scope → role
}

// NewMapper creates a mapper with global scope mapping.
func NewMapper(global map[string]config.Role) *Mapper {
	return &Mapper{global: global, tenants: make(map[string]map[string]config.Role)}
}

// NewMultiTenantMapper creates a mapper with per-tenant + global fallback.
func NewMultiTenantMapper(global map[string]config.Role, tenants map[string]config.Tenant) *Mapper {
	m := &Mapper{global: global, tenants: make(map[string]map[string]config.Role)}
	for issuer, t := range tenants {
		if len(t.ScopeMapping) > 0 {
			m.tenants[issuer] = t.ScopeMapping
		}
	}
	return m
}

// Map resolves scopes to backend roles using global mapping.
func (m *Mapper) Map(scopes []string) []string {
	return m.resolve(scopes, m.global)
}

// MapForIssuer resolves scopes using tenant-specific mapping, falling back to global.
func (m *Mapper) MapForIssuer(issuer string, scopes []string) []string {
	if tenant, ok := m.tenants[issuer]; ok {
		return m.resolve(scopes, tenant)
	}
	return m.resolve(scopes, m.global)
}

func (m *Mapper) resolve(scopes []string, mapping map[string]config.Role) []string {
	var roles []string
	seen := make(map[string]bool)
	for _, s := range scopes {
		if role, ok := mapping[s]; ok {
			for _, r := range role.BackendRoles {
				if !seen[r] {
					roles = append(roles, r)
					seen[r] = true
				}
			}
		}
	}
	return roles
}
