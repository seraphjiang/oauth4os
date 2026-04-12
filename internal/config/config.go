package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Upstream     Upstream          `yaml:"upstream"`
	Providers    []Provider        `yaml:"providers"`
	ScopeMapping map[string]Role   `yaml:"scope_mapping"`
	Listen       string            `yaml:"listen"`
}

type Upstream struct {
	Engine     string `yaml:"engine"`
	Dashboards string `yaml:"dashboards"`
}

type Provider struct {
	Name    string `yaml:"name"`
	Issuer  string `yaml:"issuer"`
	JWKSURI string `yaml:"jwks_uri"`
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
