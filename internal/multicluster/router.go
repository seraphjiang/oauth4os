// Package multicluster routes requests to multiple OpenSearch clusters
// based on path prefix or header. Single oauth4os proxy manages auth
// for N clusters with per-cluster scope mappings.
package multicluster

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
)

// Cluster represents a backend OpenSearch cluster.
type Cluster struct {
	Name     string `yaml:"name" json:"name"`
	Engine   string `yaml:"engine" json:"engine"`       // e.g. https://cluster-a:9200
	Prefix   string `yaml:"prefix" json:"prefix"`       // e.g. /cluster-a
	Default  bool   `yaml:"default" json:"default"`     // fallback cluster
}

// Config is the YAML-friendly multi-cluster configuration.
type Config struct {
	Clusters []Cluster `yaml:"clusters"`
}

// Router dispatches requests to the correct cluster proxy.
type Router struct {
	mu       sync.RWMutex
	clusters map[string]*entry // prefix → entry
	byName   map[string]*entry // name → entry
	fallback *entry
}

type entry struct {
	cluster Cluster
	proxy   *httputil.ReverseProxy
}

// NewRouter builds a router from config.
func NewRouter(cfg Config) (*Router, error) {
	r := &Router{
		clusters: make(map[string]*entry),
		byName:   make(map[string]*entry),
	}
	for _, c := range cfg.Clusters {
		if err := r.Add(c); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Add registers a cluster. Can be called at runtime via Admin API.
func (r *Router) Add(c Cluster) error {
	u, err := url.Parse(c.Engine)
	if err != nil {
		return fmt.Errorf("cluster %s: invalid engine URL: %w", c.Name, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, fmt.Sprintf(`{"error":"upstream_error","cluster":"%s","message":"%s"}`, c.Name, err.Error()), http.StatusBadGateway)
	}
	e := &entry{cluster: c, proxy: proxy}

	r.mu.Lock()
	if c.Prefix != "" {
		r.clusters[strings.TrimSuffix(c.Prefix, "/")] = e
	}
	r.byName[c.Name] = e
	if c.Default {
		r.fallback = e
	}
	r.mu.Unlock()
	return nil
}

// Remove unregisters a cluster by name.
func (r *Router) Remove(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.byName[name]
	if !ok {
		return false
	}
	delete(r.byName, name)
	if e.cluster.Prefix != "" {
		delete(r.clusters, strings.TrimSuffix(e.cluster.Prefix, "/"))
	}
	if r.fallback == e {
		r.fallback = nil
	}
	return true
}

// List returns all registered clusters.
func (r *Router) List() []Cluster {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Cluster, 0, len(r.byName))
	for _, e := range r.byName {
		out = append(out, e.cluster)
	}
	return out
}

// ServeHTTP routes the request to the matching cluster.
// Resolution order: X-Cluster header → path prefix → default cluster.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	e := r.resolve(req)
	if e == nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"no_cluster","message":"no matching cluster for request"}`, http.StatusBadGateway)
		return
	}
	// Strip prefix from path before forwarding
	if e.cluster.Prefix != "" {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, e.cluster.Prefix)
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
	}
	req.Header.Set("X-Routed-Cluster", e.cluster.Name)
	e.proxy.ServeHTTP(w, req)
}

func (r *Router) resolve(req *http.Request) *entry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Explicit header
	if name := req.Header.Get("X-Cluster"); name != "" {
		if e, ok := r.byName[name]; ok {
			return e
		}
	}
	// 2. Path prefix match
	for prefix, e := range r.clusters {
		if strings.HasPrefix(req.URL.Path, prefix) {
			return e
		}
	}
	// 3. Default
	return r.fallback
}
