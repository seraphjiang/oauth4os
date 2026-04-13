package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Edge: Load from valid YAML file
func TestEdge_LoadValidYAML(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	os.WriteFile(p, []byte("listen: :8443\nbackend: http://localhost:9200\n"), 0644)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load should succeed: %v", err)
	}
	if cfg.Listen != ":8443" {
		t.Errorf("expected :8443, got %q", cfg.Listen)
	}
}

// Edge: Load from nonexistent file fails
func TestEdge_LoadNonexistent(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Load should fail for nonexistent file")
	}
}

// Edge: Load from empty file returns defaults
func TestEdge_LoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "empty.yaml")
	os.WriteFile(p, []byte(""), 0644)
	_, err := Load(p)
	if err != nil {
		t.Fatalf("Load empty should not error: %v", err)
	}
}

// Edge: Load with invalid YAML fails
func TestEdge_LoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.yaml")
	os.WriteFile(p, []byte("{{{{not yaml"), 0644)
	_, err := Load(p)
	if err == nil {
		t.Error("Load should fail for invalid YAML")
	}
}
