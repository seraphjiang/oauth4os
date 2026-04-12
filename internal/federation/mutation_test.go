package federation

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mutation: remove cluster resolution → Resolve must find matching cluster
func TestMutation_ResolveMatch(t *testing.T) {
	r := New([]Cluster{{Name: "logs", URL: "http://logs:9200", Indices: []string{"logs-*"}}}, nil)
	url, name := r.Resolve("/logs-2024/_search")
	if name != "logs" || url == "" {
		t.Errorf("expected logs cluster, got name=%s url=%s", name, url)
	}
}

// Mutation: remove fallback → unknown index must return empty
func TestMutation_ResolveNoMatch(t *testing.T) {
	r := New([]Cluster{{Name: "logs", URL: "http://logs:9200", Indices: []string{"logs-*"}}}, nil)
	_, name := r.Resolve("/unknown-index/_search")
	if name == "logs" {
		t.Error("unknown index should not match logs cluster")
	}
}

// Mutation: remove ClusterNames → must return all cluster names
func TestMutation_ClusterNames(t *testing.T) {
	r := New([]Cluster{
		{Name: "a", URL: "http://a:9200"},
		{Name: "b", URL: "http://b:9200"},
	}, nil)
	names := r.ClusterNames()
	if len(names) != 2 {
		t.Errorf("expected 2 cluster names, got %d", len(names))
	}
}

// Mutation: remove glob matching → wildcard patterns must work
func TestMutation_GlobMatch(t *testing.T) {
	if !globMatch("logs-*", "logs-2024") {
		t.Error("logs-* should match logs-2024")
	}
	if globMatch("logs-*", "metrics-2024") {
		t.Error("logs-* should not match metrics-2024")
	}
}

// Mutation: remove Route proxy → must forward to resolved cluster
func TestMutation_RouteForwards(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Cluster", "test")
		w.WriteHeader(200)
	}))
	defer backend.Close()
	r := New([]Cluster{{Name: "test", URL: backend.URL, Patterns: []string{"logs-*"}}}, nil)
	handler := r.Route(httptest.NewRequest("GET", "/logs-2024/_search", nil))
	if handler == nil {
		t.Fatal("Route must return handler for matching index")
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/logs-2024/_search", nil))
	if w.Header().Get("X-Cluster") != "test" {
		t.Error("Route must forward to resolved cluster")
	}
}
