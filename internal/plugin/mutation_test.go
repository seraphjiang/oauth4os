package plugin

import (
	"net/http/httptest"
	"testing"
)

// Mutation: remove Register → Authorize must use registered plugins
func TestMutation_RegisterAndAuthorize(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&denyAll{name: "deny"})
	r := httptest.NewRequest("GET", "/", nil)
	err := reg.Authorize(r, map[string]interface{}{"sub": "user"})
	if err == nil {
		t.Error("deny plugin must reject")
	}
}

// Mutation: remove List → must return registered plugin names
func TestMutation_ListPlugins(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&denyAll{name: "test-plugin"})
	names := reg.List()
	if len(names) == 0 {
		t.Error("List must return registered plugins")
	}
}

type denyAll struct{ name string }

func (d *denyAll) Name() string { return d.name }
func (d *denyAll) Authorize(_ *http.Request, _ map[string]interface{}) error {
	return fmt.Errorf("denied")
}
