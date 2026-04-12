package secrets

import (
	"os"
	"testing"
)

// Mutation: remove env resolution → env:VAR must resolve from environment
func TestMutation_EnvResolve(t *testing.T) {
	os.Setenv("TEST_SECRET_XYZ", "my-secret")
	defer os.Unsetenv("TEST_SECRET_XYZ")
	r := New()
	val, err := r.Resolve("env:TEST_SECRET_XYZ")
	if err != nil {
		t.Fatal(err)
	}
	if val != "my-secret" {
		t.Errorf("expected 'my-secret', got %q", val)
	}
}

// Mutation: remove file resolution → file:path must read from file
func TestMutation_FileResolve(t *testing.T) {
	f, _ := os.CreateTemp("", "secret-*")
	f.WriteString("file-secret")
	f.Close()
	defer os.Remove(f.Name())
	r := New()
	val, err := r.Resolve("file:" + f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if val != "file-secret" {
		t.Errorf("expected 'file-secret', got %q", val)
	}
}

// Mutation: remove plain passthrough → non-ref strings returned as-is
func TestMutation_PlainPassthrough(t *testing.T) {
	r := New()
	val, err := r.Resolve("plain-value")
	if err != nil {
		t.Fatal(err)
	}
	if val != "plain-value" {
		t.Errorf("expected 'plain-value', got %q", val)
	}
}
