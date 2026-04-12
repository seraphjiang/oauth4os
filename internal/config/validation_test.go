package config

import (
	"strings"
	"testing"
)

func TestValidate_MissingUpstream(t *testing.T) {
	c := &Config{}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "upstream.engine") {
		t.Fatalf("expected upstream.engine error, got %v", err)
	}
}

func TestValidate_DefaultListen(t *testing.T) {
	c := &Config{Upstream: Upstream{Engine: "http://localhost:9200"}}
	c.Validate()
	if c.Listen != ":8443" {
		t.Fatalf("expected default listen :8443, got %q", c.Listen)
	}
}

func TestValidate_InvalidEngineURL(t *testing.T) {
	c := &Config{Upstream: Upstream{Engine: "://bad"}}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "upstream.engine") {
		t.Fatalf("expected URL error, got %v", err)
	}
}

func TestValidate_DuplicateProvider(t *testing.T) {
	c := &Config{
		Upstream:  Upstream{Engine: "http://localhost:9200"},
		Providers: []Provider{{Name: "a", Issuer: "http://a"}, {Name: "a", Issuer: "http://b"}},
	}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestValidate_EmptyScopeKey(t *testing.T) {
	c := &Config{
		Upstream:     Upstream{Engine: "http://localhost:9200"},
		ScopeMapping: map[string]Role{"": {}},
	}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "empty scope") {
		t.Fatalf("expected empty scope error, got %v", err)
	}
}

func TestValidate_TLSMissingCert(t *testing.T) {
	c := &Config{
		Upstream: Upstream{Engine: "http://localhost:9200"},
		TLS:      TLSConfig{Enabled: true},
	}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "cert_file") {
		t.Fatalf("expected TLS cert error, got %v", err)
	}
}

func TestValidate_SigV4MissingRegion(t *testing.T) {
	c := &Config{
		Upstream: Upstream{Engine: "https://x.aoss.amazonaws.com", SigV4: &SigV4Config{}},
	}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "sigv4.region") {
		t.Fatalf("expected sigv4.region error, got %v", err)
	}
}

func TestValidate_SigV4InvalidService(t *testing.T) {
	c := &Config{
		Upstream: Upstream{Engine: "https://x.aoss.amazonaws.com", SigV4: &SigV4Config{Region: "us-west-2", Service: "invalid"}},
	}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "sigv4.service") {
		t.Fatalf("expected sigv4.service error, got %v", err)
	}
}

func TestValidate_SigV4DefaultService(t *testing.T) {
	c := &Config{
		Upstream: Upstream{Engine: "https://x.aoss.amazonaws.com", SigV4: &SigV4Config{Region: "us-west-2"}},
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Upstream.SigV4.Service != "es" {
		t.Fatalf("expected default service 'es', got %q", c.Upstream.SigV4.Service)
	}
}

func TestValidate_AOSSRequiresHTTPS(t *testing.T) {
	c := &Config{
		Upstream: Upstream{Engine: "http://x.aoss.amazonaws.com", SigV4: &SigV4Config{Region: "us-west-2", Service: "aoss"}},
	}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("expected AOSS https error, got %v", err)
	}
}

func TestValidate_ValidComplete(t *testing.T) {
	c := &Config{
		Upstream: Upstream{Engine: "https://search.us-west-2.es.amazonaws.com", SigV4: &SigV4Config{Region: "us-west-2", Service: "es"}},
		Listen:   ":8443",
		Providers: []Provider{{Name: "okta", Issuer: "https://dev.okta.com"}},
		ScopeMapping: map[string]Role{"read:logs": {BackendUser: "reader"}},
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("valid config should pass: %v", err)
	}
}
