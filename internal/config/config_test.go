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

func TestLoadWithTenants(t *testing.T) {
	content := `
upstream:
  engine: http://localhost:9200
tenants:
  "https://tenant-a.example.com":
    scope_mapping:
      "read:logs":
        backend_roles: [tenant_a_read]
    cedar_policies:
      - 'permit(*, *, *);'
rate_limits:
  "read:logs": 100
  "admin": 10
listen: ":8443"
`
	f, _ := os.CreateTemp("", "oauth4os-tenant-*.yaml")
	f.WriteString(content)
	f.Close()
	defer os.Remove(f.Name())

	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	tenant, ok := cfg.Tenants["https://tenant-a.example.com"]
	if !ok {
		t.Fatal("missing tenant")
	}
	if len(tenant.CedarPolicies) != 1 {
		t.Errorf("expected 1 cedar policy, got %d", len(tenant.CedarPolicies))
	}
	if cfg.RateLimits["read:logs"] != 100 {
		t.Errorf("expected rate_limit 100, got %d", cfg.RateLimits["read:logs"])
	}
}
