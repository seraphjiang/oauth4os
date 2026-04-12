package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Upstream     Upstream          `yaml:"upstream"`
	Providers    []Provider        `yaml:"providers"`
	ScopeMapping map[string]Role   `yaml:"scope_mapping"`
	Tenants      map[string]Tenant `yaml:"tenants"`
	Listen       string            `yaml:"listen"`
	TLS          TLSConfig         `yaml:"tls"`
	RateLimits   map[string]int    `yaml:"rate_limits"`
	IPFilter     IPFilterConfig    `yaml:"ip_filter"`
	MTLS         MTLSConfig        `yaml:"mtls"`
}

// MTLSConfig holds mutual TLS client auth settings.
type MTLSConfig struct {
	Enabled  bool                       `yaml:"enabled"`
	CAFile   string                     `yaml:"ca_file"`
	Clients  map[string]*MTLSClientEntry `yaml:"clients"`
}

// MTLSClientEntry maps a cert CN/SAN to identity.
type MTLSClientEntry struct {
	ClientID string   `yaml:"client_id"`
	Scopes   []string `yaml:"scopes"`
}

// IPFilterConfig holds IP allowlist/denylist rules.
type IPFilterConfig struct {
	Global  *IPFilterRule            `yaml:"global,omitempty"`
	Clients map[string]*IPFilterRule `yaml:"clients,omitempty"`
}

// IPFilterRule holds CIDR lists.
type IPFilterRule struct {
	Allow []string `yaml:"allow,omitempty"`
	Deny  []string `yaml:"deny,omitempty"`
}

// Tenant holds per-provider scope mapping and Cedar policies.
type Tenant struct {
	ScopeMapping  map[string]Role `yaml:"scope_mapping"`
	CedarPolicies []string        `yaml:"cedar_policies"`
}

type TLSConfig struct {
	Enabled            bool   `yaml:"enabled"`
	CertFile           string `yaml:"cert_file"`
	KeyFile            string `yaml:"key_file"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"` // for self-signed upstream certs
}

type Upstream struct {
	Engine     string `yaml:"engine"`
	Dashboards string `yaml:"dashboards"`
}

type Provider struct {
	Name     string   `yaml:"name"`
	Issuer   string   `yaml:"issuer"`
	JWKSURI  string   `yaml:"jwks_uri"`
	Audience []string `yaml:"audience"` // expected aud values; if empty, aud not checked
}

type Role struct {
	BackendUser  string   `yaml:"backend_user"`
	BackendRoles []string `yaml:"backend_roles"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
