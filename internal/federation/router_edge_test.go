package federation

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentResolve(t *testing.T) {
	r := New([]Cluster{
		{Name: "us-west", URL: "http://us-west:9200", Indices: []string{"logs-*"}},
		{Name: "eu-central", URL: "http://eu:9200", Indices: []string{"metrics-*"}},
	}, nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			var path string
			if n%2 == 0 {
				path = "/logs-app/_search"
			} else {
				path = "/metrics-cpu/_search"
			}
			url, name := r.Resolve(path)
			if url == "" || name == "" {
				t.Errorf("resolve failed for %s", path)
			}
		}(i)
	}
	wg.Wait()
}

func TestResolveDefaultFallback(t *testing.T) {
	r := New([]Cluster{
		{Name: "us-west", URL: "http://us-west:9200", Indices: []string{"logs-*"}},
	}, nil)

	url, name := r.Resolve("/unknown-index/_search")
	// Router falls back to first cluster as default
	if url == "" {
		t.Fatal("expected default fallback")
	}
	_ = name
}

func TestResolveFirstMatchWins(t *testing.T) {
	r := New([]Cluster{
		{Name: "primary", URL: "http://primary:9200", Indices: []string{"logs-*"}},
		{Name: "secondary", URL: "http://secondary:9200", Indices: []string{"logs-*"}},
	}, nil)

	_, name := r.Resolve("/logs-app/_search")
	if name != "primary" {
		t.Fatalf("expected primary (first match), got %s", name)
	}
}

func TestClusterNamesOrder(t *testing.T) {
	clusters := make([]Cluster, 5)
	for i := range clusters {
		clusters[i] = Cluster{Name: fmt.Sprintf("c%d", i), URL: fmt.Sprintf("http://c%d:9200", i), Indices: []string{"*"}}
	}
	r := New(clusters, nil)
	names := r.ClusterNames()
	if len(names) != 5 {
		t.Fatalf("expected 5 clusters, got %d", len(names))
	}
	for i, n := range names {
		if n != fmt.Sprintf("c%d", i) {
			t.Fatalf("expected c%d at position %d, got %s", i, i, n)
		}
	}
}
