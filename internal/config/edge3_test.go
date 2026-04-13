package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEdge_LoadWithProviders(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	os.WriteFile(p, []byte("listen: :8443\nproviders:\n  - name: github\n    issuer: https://github.com\n"), 0644)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Providers) == 0 {
		t.Error("should parse providers")
	}
}

func TestEdge_LoadWithScopeMapping(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	os.WriteFile(p, []byte("listen: :8443\nscope_mapping:\n  admin:\n    backend_user: admin\n    backend_roles: [all_access]\n"), 0644)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.ScopeMapping) == 0 {
		t.Error("should parse scope mapping")
	}
}
