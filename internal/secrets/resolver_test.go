package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLiteral(t *testing.T) {
	r := New()
	v, err := r.Resolve("plain-secret")
	if err != nil || v != "plain-secret" {
		t.Errorf("got %q, %v", v, err)
	}
}

func TestEnvBackend(t *testing.T) {
	t.Setenv("TEST_SECRET_123", "hunter2")
	r := New()
	v, err := r.Resolve("env:TEST_SECRET_123")
	if err != nil || v != "hunter2" {
		t.Errorf("got %q, %v", v, err)
	}
}

func TestEnvBackend_Missing(t *testing.T) {
	r := New()
	_, err := r.Resolve("env:NONEXISTENT_VAR_XYZ")
	if err == nil {
		t.Error("expected error for missing env var")
	}
}

func TestFileBackend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")
	os.WriteFile(path, []byte("file-secret\n"), 0600)

	r := New()
	v, err := r.Resolve("file:" + path)
	if err != nil {
		t.Fatal(err)
	}
	if v != "file-secret" {
		t.Errorf("got %q, want file-secret", v)
	}
}

func TestFileBackend_Missing(t *testing.T) {
	r := New()
	_, err := r.Resolve("file:/nonexistent/path")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestUnknownBackend(t *testing.T) {
	r := New()
	_, err := r.Resolve("aws:my-secret")
	if err == nil {
		t.Error("expected error for unregistered aws backend")
	}
}

func TestResolveAll(t *testing.T) {
	t.Setenv("DB_PASS", "s3cret")
	r := New()
	m := map[string]string{
		"password": "env:DB_PASS",
		"literal":  "plain",
	}
	if err := r.ResolveAll(m); err != nil {
		t.Fatal(err)
	}
	if m["password"] != "s3cret" {
		t.Errorf("password = %q", m["password"])
	}
	if m["literal"] != "plain" {
		t.Errorf("literal = %q", m["literal"])
	}
}

func TestParseRef_EdgeCases(t *testing.T) {
	cases := []struct {
		input  string
		scheme string
		ok     bool
	}{
		{"", "", false},
		{"noscheme", "", false},
		{"env:", "", false},
		{"http://example.com", "", false},
		{"env:VAR", "env", true},
		{"file:/tmp/x", "file", true},
	}
	for _, tc := range cases {
		s, _, ok := parseRef(tc.input)
		if ok != tc.ok || (ok && s != tc.scheme) {
			t.Errorf("parseRef(%q) = %q, %v; want %q, %v", tc.input, s, ok, tc.scheme, tc.ok)
		}
	}
}
