package federation

import (
	"net/http"
	"testing"
)

func TestRouterResolve(t *testing.T) {
	clusters := []Cluster{
		{Name: "logs", URL: "https://logs:9200", Indices: []string{"logs-*", "audit-*"}},
		{Name: "metrics", URL: "https://metrics:9200", Indices: []string{"metrics-*"}},
		{Name: "default", URL: "https://general:9200", Indices: []string{"*"}},
	}
	r := New(clusters, nil)

	tests := []struct {
		path string
		url  string
		name string
	}{
		{"/logs-2025/_search", "https://logs:9200", "logs"},
		{"/audit-trail/_search", "https://logs:9200", "logs"},
		{"/metrics-cpu/_search", "https://metrics:9200", "metrics"},
		{"/users/_search", "https://general:9200", "default"},
		{"/_cat/health", "https://general:9200", "default"},
		{"/", "https://general:9200", "default"},
	}

	for _, tt := range tests {
		url, name := r.Resolve(tt.path)
		if url != tt.url {
			t.Errorf("Resolve(%q) url = %q, want %q", tt.path, url, tt.url)
		}
		if name != tt.name {
			t.Errorf("Resolve(%q) name = %q, want %q", tt.path, name, tt.name)
		}
	}
}

func TestClusterNames(t *testing.T) {
	clusters := []Cluster{
		{Name: "a", URL: "http://a:9200"},
		{Name: "b", URL: "http://b:9200"},
	}
	r := New(clusters, nil)
	names := r.ClusterNames()
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Errorf("ClusterNames() = %v, want [a b]", names)
	}
}

func TestRouteNilWhenNoMatch(t *testing.T) {
	r := New([]Cluster{{Name: "prod", URL: "http://prod:9200", Indices: []string{"prod-*"}}}, nil)
	req, _ := http.NewRequest("GET", "/dev-logs/_search", nil)
	if handler := r.Route(req); handler != nil {
		t.Fatal("expected nil handler for non-matching index")
	}
}

func TestExtractIndex(t *testing.T) {
	tests := []struct{ path, want string }{
		{"/logs-2024/_search", "logs-2024"},
		{"/my-index/_doc/1", "my-index"},
		{"/_cluster/health", ""},
		{"/", ""},
	}
	for _, tt := range tests {
		got := extractIndex(tt.path)
		if got != tt.want {
			t.Errorf("extractIndex(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestGlobMatch(t *testing.T) {
	tests := []struct{ pattern, value string; want bool }{
		{"logs-*", "logs-2024", true},
		{"logs-*", "metrics-2024", false},
		{"*", "anything", true},
		{"exact", "exact", true},
		{"exact", "other", false},
	}
	for _, tt := range tests {
		got := globMatch(tt.pattern, tt.value)
		if got != tt.want {
			t.Errorf("globMatch(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
		}
	}
}
