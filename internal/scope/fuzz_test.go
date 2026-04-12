package scope

import (
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

// FuzzMapper tests scope mapper with random scope strings.
// Run: go test -fuzz=FuzzMapper -fuzztime=30s ./internal/scope/
func FuzzMapper(f *testing.F) {
	f.Add("read:logs")
	f.Add("write:logs read:metrics admin")
	f.Add("")
	f.Add(string(make([]byte, 5000)))
	f.Add("read:logs\x00write:logs")

	m := NewMapper(map[string]config.Role{
		"read:logs":  {BackendRoles: []string{"logs_read"}},
		"write:logs": {BackendRoles: []string{"logs_write"}},
		"admin":      {BackendRoles: []string{"all_access"}},
	})

	f.Fuzz(func(t *testing.T, scopeStr string) {
		// Split on spaces like the real code does
		var scopes []string
		start := 0
		for i := 0; i < len(scopeStr); i++ {
			if scopeStr[i] == ' ' {
				if i > start {
					scopes = append(scopes, scopeStr[start:i])
				}
				start = i + 1
			}
		}
		if start < len(scopeStr) {
			scopes = append(scopes, scopeStr[start:])
		}
		// Must not panic
		m.Map(scopes)
	})
}
