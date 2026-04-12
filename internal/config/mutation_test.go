package config

import (
	"os"
	"testing"
)

// Mutation: remove file read → Load must fail on missing file
func TestMutation_LoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Load must fail on missing file")
	}
}

// Mutation: remove YAML parsing → Load must fail on invalid YAML
func TestMutation_LoadInvalidYAML(t *testing.T) {
	f, _ := os.CreateTemp("", "cfg-*.yaml")
	f.WriteString("{{invalid yaml")
	f.Close()
	defer os.Remove(f.Name())
	_, err := Load(f.Name())
	if err == nil {
		t.Error("Load must fail on invalid YAML")
	}
}

// Mutation: remove validation → Validate must catch missing upstream
func TestMutation_ValidateMissingUpstream(t *testing.T) {
	c := &Config{}
	if err := c.Validate(); err == nil {
		t.Error("Validate must fail when upstream is not configured")
	}
}
