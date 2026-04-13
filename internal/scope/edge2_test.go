package scope

import (
	"sync"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

func TestEdge_ConcurrentMap(t *testing.T) {
	m := NewMapper(map[string]config.Role{
		"admin": {BackendUser: "admin", BackendRoles: []string{"all_access"}},
	})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Map([]string{"admin", "read"})
		}()
	}
	wg.Wait()
}

func TestEdge_MapForUnknownIssuer(t *testing.T) {
	m := NewMultiTenantMapper(nil, map[string]config.Tenant{
		"known": {ScopeMapping: map[string]config.Role{
			"read": {BackendUser: "r", BackendRoles: []string{"readall"}},
		}},
	})
	got := m.MapForIssuer("unknown-issuer", []string{"read"})
	// Unknown issuer falls back to global — should not panic
	_ = got
}
