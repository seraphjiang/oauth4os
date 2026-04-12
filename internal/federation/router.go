// Package federation routes requests to the correct OpenSearch cluster
// based on index prefix matching.
//
// Config example:
//
//	clusters:
//	  logs:
//	    engine: https://logs-cluster:9200
//	    prefixes: ["logs-*", "audit-*"]
//	  metrics:
//	    engine: https://metrics-cluster:9200
//	    prefixes: ["metrics-*", ".ds-metrics-*"]
//	  default:
//	    engine: https://general-cluster:9200
//	    prefixes: ["*"]
package federation

import (
	"fmt"
	"path"
	"strings"

	"github.com/seraphjiang/oauth4os/internal/config"
)

// Router resolves which cluster handles a request based on the URL path.
type Router struct {
	clusters []entry
	fallback string // default cluster engine URL
}

type entry struct {
	name     string
	engine   string
	prefixes []string
}

// NewRouter builds a router from cluster config. Last cluster with prefix "*" is fallback.
func NewRouter(clusters map[string]config.Cluster, defaultEngine string) *Router {
	r := &Router{fallback: defaultEngine}
	for name, c := range clusters {
		e := entry{name: name, engine: c.Engine, prefixes: c.Prefixes}
		for _, p := range c.Prefixes {
			if p == "*" {
				r.fallback = c.Engine
			}
		}
		r.clusters = append(r.clusters, e)
	}
	return r
}

// Resolve returns the upstream engine URL for a given request path.
func (r *Router) Resolve(urlPath string) (engine string, cluster string) {
	index := extractIndex(urlPath)
	if index == "" {
		return r.fallback, "default"
	}
	for _, e := range r.clusters {
		for _, prefix := range e.prefixes {
			if matched, _ := path.Match(prefix, index); matched {
				return e.engine, e.name
			}
		}
	}
	return r.fallback, "default"
}

// extractIndex pulls the index name from an OpenSearch URL path.
// e.g. "/logs-2025/_search" → "logs-2025", "/_cat/health" → ""
func extractIndex(p string) string {
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return ""
	}
	parts := strings.SplitN(p, "/", 2)
	idx := parts[0]
	// Skip internal/API paths
	if strings.HasPrefix(idx, "_") || strings.HasPrefix(idx, ".kibana") {
		return ""
	}
	return idx
}

// ListClusters returns all configured cluster names and endpoints.
func (r *Router) ListClusters() map[string]string {
	result := make(map[string]string, len(r.clusters))
	for _, e := range r.clusters {
		result[e.name] = e.engine
	}
	return result
}

// String returns a human-readable routing table.
func (r *Router) String() string {
	var sb strings.Builder
	for _, e := range r.clusters {
		fmt.Fprintf(&sb, "  %s → %s (prefixes: %s)\n", e.name, e.engine, strings.Join(e.prefixes, ", "))
	}
	fmt.Fprintf(&sb, "  fallback → %s\n", r.fallback)
	return sb.String()
}
