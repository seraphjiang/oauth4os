// Package secrets resolves secret references in configuration.
// Supports env vars (default), file paths, and AWS Secrets Manager.
//
// Config values prefixed with a scheme are resolved:
//
//	env:MY_VAR          → os.Getenv("MY_VAR")
//	file:/path/to/secret → reads file contents (trimmed)
//	aws:secret-name      → placeholder for AWS Secrets Manager (v1.2.0)
//
// Unprefixed values are returned as-is (literal secrets).
package secrets

import (
	"fmt"
	"os"
	"strings"
)

// Backend resolves a secret reference to its plaintext value.
type Backend interface {
	Resolve(ref string) (string, error)
}

// Resolver resolves secret references using the configured backend.
type Resolver struct {
	backends map[string]Backend
}

// New creates a resolver with env and file backends.
func New() *Resolver {
	return &Resolver{
		backends: map[string]Backend{
			"env":  envBackend{},
			"file": fileBackend{},
		},
	}
}

// Resolve resolves a secret reference. If ref has a scheme prefix
// (env:, file:), the corresponding backend is used. Otherwise
// the value is returned as a literal.
func (r *Resolver) Resolve(ref string) (string, error) {
	scheme, key, ok := parseRef(ref)
	if !ok {
		return ref, nil // literal value
	}
	b, exists := r.backends[scheme]
	if !exists {
		return "", fmt.Errorf("unknown secrets backend: %s", scheme)
	}
	return b.Resolve(key)
}

// ResolveAll resolves a map of secret references in place.
func (r *Resolver) ResolveAll(m map[string]string) error {
	for k, v := range m {
		resolved, err := r.Resolve(v)
		if err != nil {
			return fmt.Errorf("resolving %s: %w", k, err)
		}
		m[k] = resolved
	}
	return nil
}

func parseRef(ref string) (scheme, key string, ok bool) {
	i := strings.Index(ref, ":")
	if i < 1 || i == len(ref)-1 {
		return "", "", false
	}
	s := ref[:i]
	if s == "env" || s == "file" || s == "aws" {
		return s, ref[i+1:], true
	}
	return "", "", false
}

// envBackend resolves secrets from environment variables.
type envBackend struct{}

func (envBackend) Resolve(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("env var %s not set", key)
	}
	return v, nil
}

// fileBackend resolves secrets from file contents.
type fileBackend struct{}

func (fileBackend) Resolve(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading secret file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}
