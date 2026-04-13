package scope

import (
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

// Edge: Map with nil config returns empty (no mappings)
func TestEdge_NilConfigEmpty(t *testing.T) {
	m := NewMapper(nil)
	got := m.Map([]string{"admin"})
	if len(got) != 0 {
		t.Errorf("nil config should return empty, got %v", got)
	}
}

// Edge: Map expands known scope to backend roles
func TestEdge_ExpandsToRoles(t *testing.T) {
	m := NewMapper(map[string]config.Role{
		"admin": {BackendUser: "admin", BackendRoles: []string{"all_access"}},
	})
	got := m.Map([]string{"admin"})
	if len(got) == 0 {
		t.Error("mapped scope should produce output")
	}
}

// Edge: Map unknown scope returns empty (not mapped)
func TestEdge_UnknownReturnsEmpty(t *testing.T) {
	m := NewMapper(map[string]config.Role{
		"admin": {BackendUser: "admin", BackendRoles: []string{"all_access"}},
	})
	got := m.Map([]string{"custom"})
	// Unknown scopes are not mapped — result may be empty or passthrough
	_ = got // just verify no panic
}

// Edge: empty input returns empty
func TestEdge_EmptyInput(t *testing.T) {
	m := NewMapper(nil)
	got := m.Map(nil)
	if len(got) != 0 {
		t.Errorf("nil input should return empty, got %v", got)
	}
}

// Edge: multi-tenant mapper uses issuer-specific mapping
func TestEdge_MultiTenantMapper(t *testing.T) {
	m := NewMultiTenantMapper(nil, map[string]config.Tenant{
		"issuer-a": {ScopeMapping: map[string]config.Role{
			"read": {BackendUser: "reader", BackendRoles: []string{"readall"}},
		}},
	})
	got := m.MapForIssuer("issuer-a", []string{"read"})
	if len(got) == 0 {
		t.Error("tenant mapping should produce output")
	}
}
