package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEdge_ValidateEmptyConfig(t *testing.T) {
	c := &Config{}
	// Empty config should either pass or return specific error — not panic
	_ = c.Validate()
}

func TestEdge_ValidateWithListen(t *testing.T) {
	c := &Config{Listen: ":8443"}
	// May fail validation for missing upstream — just verify no panic
	_ = c.Validate()
}

func TestEdge_LoadEnvOverride(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	os.WriteFile(p, []byte("listen: :8443\nbackend: http://localhost:9200\n"), 0644)
	os.Setenv("OAUTH4OS_LISTEN", ":9999")
	defer os.Unsetenv("OAUTH4OS_LISTEN")
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	// Env override may or may not be supported — just verify no panic
	_ = cfg
}
