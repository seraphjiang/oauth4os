package federation

import (
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

func TestRouterResolve(t *testing.T) {
	clusters := map[string]config.Cluster{
		"logs":    {Engine: "https://logs:9200", Prefixes: []string{"logs-*", "audit-*"}},
		"metrics": {Engine: "https://metrics:9200", Prefixes: []string{"metrics-*"}},
		"default": {Engine: "https://general:9200", Prefixes: []string{"*"}},
	}
	r := NewRouter(clusters, "https://fallback:9200")

	tests := []struct {
		path    string
		engine  string
		cluster string
	}{
		{"/logs-2025/_search", "https://logs:9200", "logs"},
		{"/audit-trail/_search", "https://logs:9200", "logs"},
		{"/metrics-cpu/_search", "https://metrics:9200", "metrics"},
		{"/users/_search", "https://general:9200", "default"},
		{"/_cat/health", "https://general:9200", "default"},
		{"/", "https://general:9200", "default"},
	}

	for _, tt := range tests {
		engine, cluster := r.Resolve(tt.path)
		if engine != tt.engine {
			t.Errorf("Resolve(%q) engine = %q, want %q", tt.path, engine, tt.engine)
		}
		if cluster != tt.cluster {
			t.Errorf("Resolve(%q) cluster = %q, want %q", tt.path, cluster, tt.cluster)
		}
	}
}

func TestRouterFallback(t *testing.T) {
	r := NewRouter(nil, "https://fallback:9200")
	engine, _ := r.Resolve("/anything/_search")
	if engine != "https://fallback:9200" {
		t.Errorf("expected fallback, got %q", engine)
	}
}
