package plugin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type denyPlugin struct{}

func (d denyPlugin) Authorize(*http.Request, map[string]interface{}) error {
	return errors.New("denied")
}
func (d denyPlugin) Name() string { return "deny" }

type allowPlugin struct{}

func (a allowPlugin) Authorize(*http.Request, map[string]interface{}) error { return nil }
func (a allowPlugin) Name() string                                          { return "allow" }

func TestRegistry_AllowAll(t *testing.T) {
	reg := NewRegistry()
	reg.Register(allowPlugin{})
	r := httptest.NewRequest("GET", "/", nil)
	if err := reg.Authorize(r, nil); err != nil {
		t.Errorf("expected allow, got %v", err)
	}
}

func TestRegistry_FirstDenialWins(t *testing.T) {
	reg := NewRegistry()
	reg.Register(allowPlugin{})
	reg.Register(denyPlugin{})
	r := httptest.NewRequest("GET", "/", nil)
	err := reg.Authorize(r, nil)
	if err == nil {
		t.Error("expected denial")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	reg.Register(allowPlugin{})
	reg.Register(denyPlugin{})
	names := reg.List()
	if len(names) != 2 || names[0] != "allow" || names[1] != "deny" {
		t.Errorf("List = %v", names)
	}
}

func TestRegistry_Empty(t *testing.T) {
	reg := NewRegistry()
	r := httptest.NewRequest("GET", "/", nil)
	if err := reg.Authorize(r, nil); err != nil {
		t.Errorf("empty registry should allow: %v", err)
	}
}

func TestRegistry_LoadMissing(t *testing.T) {
	reg := NewRegistry()
	err := reg.Load("/nonexistent.so")
	if err == nil {
		t.Error("expected error for missing plugin")
	}
}
