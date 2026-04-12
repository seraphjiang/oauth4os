// Package federation routes requests to the correct OpenSearch cluster
// based on index prefix matching.
package federation

import (
	"net/http"
	"path"
	"strings"
)

// Cluster defines a named OpenSearch cluster.
type Cluster struct {
	Name    string
	URL     string
	Indices []string
}

// Router resolves which cluster handles a request based on the URL path.
type Router struct {
	clusters  []Cluster
	fallback  string
	transport http.RoundTripper
}

// New builds a router from cluster list. Last cluster with "*" index is fallback.
func New(clusters []Cluster, transport http.RoundTripper) *Router {
	r := &Router{clusters: clusters, transport: transport}
	if len(clusters) > 0 {
		r.fallback = clusters[0].URL
	}
	for _, c := range clusters {
		for _, p := range c.Indices {
			if p == "*" {
				r.fallback = c.URL
			}
		}
	}
	return r
}

// Resolve returns the upstream URL and cluster name for a request path.
func (r *Router) Resolve(urlPath string) (url string, name string) {
	index := extractIndex(urlPath)
	if index == "" {
		return r.fallback, "default"
	}
	for _, c := range r.clusters {
		for _, pattern := range c.Indices {
			if matched, _ := path.Match(pattern, index); matched {
				return c.URL, c.Name
			}
		}
	}
	return r.fallback, "default"
}

// ClusterNames returns all configured cluster names.
func (r *Router) ClusterNames() []string {
	names := make([]string, len(r.clusters))
	for i, c := range r.clusters {
		names[i] = c.Name
	}
	return names
}

func extractIndex(p string) string {
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return ""
	}
	parts := strings.SplitN(p, "/", 2)
	idx := parts[0]
	if strings.HasPrefix(idx, "_") || strings.HasPrefix(idx, ".kibana") {
		return ""
	}
	return idx
}
