// Package ipfilter provides per-client IP allowlist/denylist enforcement.
package ipfilter

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// Rules maps client_id → filter config.
type Rules struct {
	clients map[string]*ClientFilter
	global  *ClientFilter // applies when no client-specific rule
}

// ClientFilter holds parsed CIDRs for one client.
type ClientFilter struct {
	Allow []*net.IPNet
	Deny  []*net.IPNet
}

// Config is the YAML-friendly representation.
type Config struct {
	Global  *FilterConfig            `yaml:"global,omitempty"`
	Clients map[string]*FilterConfig `yaml:"clients,omitempty"`
}

// FilterConfig holds raw CIDR strings.
type FilterConfig struct {
	Allow []string `yaml:"allow,omitempty"`
	Deny  []string `yaml:"deny,omitempty"`
}

// New parses config into Rules.
func New(cfg Config) (*Rules, error) {
	r := &Rules{clients: make(map[string]*ClientFilter)}
	if cfg.Global != nil {
		f, err := parseFilter(cfg.Global)
		if err != nil {
			return nil, fmt.Errorf("global: %w", err)
		}
		r.global = f
	}
	for id, fc := range cfg.Clients {
		f, err := parseFilter(fc)
		if err != nil {
			return nil, fmt.Errorf("client %s: %w", id, err)
		}
		r.clients[id] = f
	}
	return r, nil
}

// Check returns nil if the IP is allowed for the client, or an error message.
func (r *Rules) Check(clientID, remoteAddr string) error {
	ip := extractIP(remoteAddr)
	if ip == nil {
		return nil // can't parse → allow (fail open for localhost/unix)
	}
	filter := r.clients[clientID]
	if filter == nil {
		filter = r.global
	}
	if filter == nil {
		return nil
	}
	// Deny takes precedence
	for _, cidr := range filter.Deny {
		if cidr.Contains(ip) {
			return fmt.Errorf("ip %s denied for client %s", ip, clientID)
		}
	}
	// If allowlist exists, IP must match
	if len(filter.Allow) > 0 {
		for _, cidr := range filter.Allow {
			if cidr.Contains(ip) {
				return nil
			}
		}
		return fmt.Errorf("ip %s not in allowlist for client %s", ip, clientID)
	}
	return nil
}

// Middleware returns an http.Handler that enforces IP filters.
func (r *Rules) Middleware(next http.Handler, getClient func(r *http.Request) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		clientID := getClient(req)
		if err := r.Check(clientID, req.RemoteAddr); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, `{"error":"ip_denied","message":"%s"}`, err.Error())
			return
		}
		next.ServeHTTP(w, req)
	})
}

func parseFilter(fc *FilterConfig) (*ClientFilter, error) {
	f := &ClientFilter{}
	for _, s := range fc.Allow {
		_, cidr, err := net.ParseCIDR(normalizeCIDR(s))
		if err != nil {
			return nil, fmt.Errorf("invalid allow CIDR %q: %w", s, err)
		}
		f.Allow = append(f.Allow, cidr)
	}
	for _, s := range fc.Deny {
		_, cidr, err := net.ParseCIDR(normalizeCIDR(s))
		if err != nil {
			return nil, fmt.Errorf("invalid deny CIDR %q: %w", s, err)
		}
		f.Deny = append(f.Deny, cidr)
	}
	return f, nil
}

func normalizeCIDR(s string) string {
	if !strings.Contains(s, "/") {
		if strings.Contains(s, ":") {
			return s + "/128"
		}
		return s + "/32"
	}
	return s
}

func extractIP(addr string) net.IP {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	return net.ParseIP(host)
}
