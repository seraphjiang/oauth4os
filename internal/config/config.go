package config

import (
	"fmt"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Upstream       Upstream          `yaml:"upstream"`
	Clusters       map[string]Cluster `yaml:"clusters"`
	Providers      []Provider        `yaml:"providers"`
	ScopeMapping   map[string]Role   `yaml:"scope_mapping"`
	Tenants        map[string]Tenant `yaml:"tenants"`
	Listen         string            `yaml:"listen"`
	TLS            TLSConfig         `yaml:"tls"`
	RateLimits     map[string]int    `yaml:"rate_limits"`
	IPFilter       IPFilterConfig    `yaml:"ip_filter"`
	MTLS           MTLSConfig        `yaml:"mtls"`
	Webhook        WebhookConfig     `yaml:"webhook"`
	CORS           CORSConfig        `yaml:"cors"`
	SecretsBackend string            `yaml:"secrets_backend"` // env (default), file
	JWTAccessToken bool              `yaml:"jwt_access_token"` // issue signed JWTs instead of opaque tokens
	Issuer         string            `yaml:"issuer"` // JWT issuer URL (required if jwt_access_token is true)
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	Origins []string `yaml:"origins"` // allowed origins, empty = "*"
	Methods []string `yaml:"methods"` // allowed methods, empty = default set
	Headers []string `yaml:"headers"` // allowed headers, empty = default set
}

// WebhookConfig holds external webhook authorizer settings.
type WebhookConfig struct {
	URL      string            `yaml:"url"`
	Timeout  int               `yaml:"timeout_ms"`
	Headers  map[string]string `yaml:"headers"`
	FailOpen bool              `yaml:"fail_open"`
}

// Cluster defines a named OpenSearch cluster for multi-cluster federation.
type Cluster struct {
	Engine     string   `yaml:"engine"`
	Dashboards string   `yaml:"dashboards,omitempty"`
	Prefixes   []string `yaml:"prefixes"`
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
	SigV4      *SigV4Config `yaml:"sigv4,omitempty"` // for AOSS
}

// SigV4Config enables AWS SigV4 request signing for AOSS/managed OpenSearch.
type SigV4Config struct {
	Region  string `yaml:"region"`
	Service string `yaml:"service"` // "aoss" for Serverless, "es" for managed
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

// Validate checks required fields and returns the first error found.
func (c *Config) Validate() error {
	if c.Upstream.Engine == "" && len(c.Clusters) == 0 {
		return fmt.Errorf("upstream.engine is required (or configure clusters)")
	}
	if c.Listen == "" {
		c.Listen = ":8443"
	}
	if c.Upstream.Engine != "" {
		if _, err := url.Parse(c.Upstream.Engine); err != nil {
			return fmt.Errorf("upstream.engine: invalid URL: %w", err)
		}
	}
	if c.Upstream.Dashboards != "" {
		if _, err := url.Parse(c.Upstream.Dashboards); err != nil {
			return fmt.Errorf("upstream.dashboards: invalid URL: %w", err)
		}
	}
	seen := make(map[string]bool)
	for _, p := range c.Providers {
		if p.Issuer == "" {
			return fmt.Errorf("provider %q: issuer is required", p.Name)
		}
		if seen[p.Name] {
			return fmt.Errorf("duplicate provider name: %q", p.Name)
		}
		seen[p.Name] = true
	}
	for scope := range c.ScopeMapping {
		if scope == "" {
			return fmt.Errorf("scope_mapping: empty scope key")
		}
	}
	if c.TLS.Enabled && (c.TLS.CertFile == "" || c.TLS.KeyFile == "") {
		return fmt.Errorf("tls: cert_file and key_file required when tls.enabled=true")
	}
	// AOSS / SigV4 validation
	if c.Upstream.SigV4 != nil {
		if c.Upstream.SigV4.Region == "" {
			return fmt.Errorf("upstream.sigv4.region is required when sigv4 is configured")
		}
		svc := c.Upstream.SigV4.Service
		if svc != "" && svc != "aoss" && svc != "es" {
			return fmt.Errorf("upstream.sigv4.service must be \"aoss\" or \"es\", got %q", svc)
		}
		if svc == "" {
			c.Upstream.SigV4.Service = "es" // default
		}
		// Validate AOSS endpoint format
		if svc == "aoss" && c.Upstream.Engine != "" {
			u, _ := url.Parse(c.Upstream.Engine)
			if u != nil && u.Scheme != "https" {
				return fmt.Errorf("upstream.engine: AOSS requires https, got %q", u.Scheme)
			}
		}
	}
	return nil
}
