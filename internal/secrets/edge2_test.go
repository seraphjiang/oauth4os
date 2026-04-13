package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEdge_ResolveFileValid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "secret.txt")
	os.WriteFile(p, []byte("my-secret-value"), 0600)
	r := New()
	v, err := r.Resolve("file:" + p)
	if err != nil {
		t.Fatal(err)
	}
	if v != "my-secret-value" {
		t.Errorf("expected 'my-secret-value', got %q", v)
	}
}

func TestEdge_ResolveAllMixed(t *testing.T) {
	os.Setenv("TEST_MIX_KEY", "from-env")
	defer os.Unsetenv("TEST_MIX_KEY")
	r := New()
	m := map[string]string{
		"env_val":   "env:TEST_MIX_KEY",
		"plain_val": "literal",
	}
	if err := r.ResolveAll(m); err != nil {
		t.Fatal(err)
	}
	if m["env_val"] != "from-env" {
		t.Errorf("env should resolve, got %q", m["env_val"])
	}
	if m["plain_val"] != "literal" {
		t.Errorf("literal should passthrough, got %q", m["plain_val"])
	}
}
