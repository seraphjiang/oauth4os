// Package ipfilter provides per-client IP allowlist/denylist filtering.
package ipfilter

import (
	"net"
	"strings"
)

// Rule defines IP restrictions for a client.
type Rule struct {
	Allow []string // CIDRs or IPs; if non-empty, only these are allowed
	Deny  []string // CIDRs or IPs; checked first (deny wins)
}

// Filter evaluates IP rules per client.
type Filter struct {
	rules map[string]*parsedRule // client_id → rule
}

type parsedRule struct {
	allowNets []*net.IPNet
	denyNets  []*net.IPNet
}

// New creates a filter from config. Key is client_id or "*" for global.
func New(rules map[string]Rule) *Filter {
	f := &Filter{rules: make(map[string]*parsedRule, len(rules))}
	for client, r := range rules {
		f.rules[client] = parseRule(r)
	}
	return f
}

// Check returns true if the IP is allowed for the given client.
func (f *Filter) Check(clientID, remoteAddr string) bool {
	ip := extractIP(remoteAddr)
	if ip == nil {
		return false
	}

	// Check client-specific rules first, then global
	for _, key := range []string{clientID, "*"} {
		rule, ok := f.rules[key]
		if !ok {
			continue
		}
		// Deny takes precedence
		for _, n := range rule.denyNets {
			if n.Contains(ip) {
				return false
			}
		}
		// If allowlist exists, IP must match
		if len(rule.allowNets) > 0 {
			allowed := false
			for _, n := range rule.allowNets {
				if n.Contains(ip) {
					allowed = true
					break
				}
			}
			return allowed
		}
	}
	return true // no rules = allow
}

func parseRule(r Rule) *parsedRule {
	return &parsedRule{
		allowNets: parseCIDRs(r.Allow),
		denyNets:  parseCIDRs(r.Deny),
	}
}

func parseCIDRs(list []string) []*net.IPNet {
	var nets []*net.IPNet
	for _, s := range list {
		s = strings.TrimSpace(s)
		if !strings.Contains(s, "/") {
			// Bare IP → /32 or /128
			if strings.Contains(s, ":") {
				s += "/128"
			} else {
				s += "/32"
			}
		}
		_, n, err := net.ParseCIDR(s)
		if err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}

func extractIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return net.ParseIP(host)
}
