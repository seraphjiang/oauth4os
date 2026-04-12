package scope

import "github.com/seraphjiang/oauth4os/internal/config"

type Mapper struct {
	mapping map[string]config.Role
}

func NewMapper(mapping map[string]config.Role) *Mapper {
	return &Mapper{mapping: mapping}
}

func (m *Mapper) Map(scopes []string) []string {
	var roles []string
	seen := make(map[string]bool)
	for _, s := range scopes {
		if role, ok := m.mapping[s]; ok {
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
