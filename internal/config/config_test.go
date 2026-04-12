package config

import (
	"os"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	content := `
upstream:
  engine: http://localhost:9200
  dashboards: http://localhost:5601
providers:
  - name: test
    issuer: https://issuer.example.com
    jwks_uri: auto
scope_mapping:
  "read:logs-*":
    backend_user: reader
    backend_roles: [logs_read]
listen: ":9090"
tls:
  enabled: true
  cert_file: /tmp/cert.pem
  key_file: /tmp/key.pem
  insecure_skip_verify: true
`
	f, _ := os.CreateTemp("", "oauth4os-*.yaml")
	f.WriteString(content)
	f.Close()
	defer os.Remove(f.Name())

	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Upstream.Engine != "http://localhost:9200" {
		t.Errorf("engine = %s", cfg.Upstream.Engine)
	}
	if cfg.Listen != ":9090" {
		t.Errorf("listen = %s", cfg.Listen)
	}
	if !cfg.TLS.Enabled {
		t.Error("expected TLS enabled")
	}
	if !cfg.TLS.InsecureSkipVerify {
		t.Error("expected insecure_skip_verify")
	}
	if len(cfg.Providers) != 1 || cfg.Providers[0].Name != "test" {
		t.Errorf("providers = %+v", cfg.Providers)
	}
	role, ok := cfg.ScopeMapping["read:logs-*"]
	if !ok || role.BackendUser != "reader" {
		t.Errorf("scope_mapping = %+v", cfg.ScopeMapping)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	f, _ := os.CreateTemp("", "oauth4os-bad-*.yaml")
	f.WriteString("{{invalid yaml")
	f.Close()
	defer os.Remove(f.Name())

	_, err := Load(f.Name())
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
