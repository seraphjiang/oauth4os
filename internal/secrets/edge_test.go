package secrets

import (
	"os"
	"testing"
)

func TestEdge_ResolveEnvVar(t *testing.T) {
	os.Setenv("TEST_SECRET_EDGE", "my-secret")
	defer os.Unsetenv("TEST_SECRET_EDGE")
	r := New()
	v, err := r.Resolve("env:TEST_SECRET_EDGE")
	if err != nil {
		t.Fatal(err)
	}
	if v != "my-secret" {
		t.Errorf("expected 'my-secret', got %q", v)
	}
}

func TestEdge_ResolveLiteral(t *testing.T) {
	r := New()
	v, err := r.Resolve("plain-text-value")
	if err != nil {
		t.Fatal(err)
	}
	if v != "plain-text-value" {
		t.Errorf("expected literal passthrough, got %q", v)
	}
}

func TestEdge_ResolveFileMissing(t *testing.T) {
	r := New()
	_, err := r.Resolve("file:/nonexistent/secret")
	if err == nil {
		t.Error("missing file should error")
	}
}

func TestEdge_ResolveEnvMissing(t *testing.T) {
	os.Unsetenv("DEFINITELY_NOT_SET_12345")
	r := New()
	_, err := r.Resolve("env:DEFINITELY_NOT_SET_12345")
	if err == nil {
		t.Error("missing env var should error")
	}
}
