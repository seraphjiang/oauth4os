// Package cedar implements a lightweight Cedar-like policy evaluator for oauth4os.
//
// Supports a subset of Cedar syntax for OpenSearch access control:
//
//	permit(principal, action, resource)
//	  when { principal.scope == "read:logs-*" }
//	  unless { resource.index == ".opendistro_security" };
//
// Policies are loaded from .cedar files or embedded in config.yaml.
package cedar

import (
	"fmt"
	"strings"
	"sync"
)

// Effect is the policy decision.
type Effect string

const (
	Permit Effect = "permit"
	Forbid Effect = "forbid"
)

// Policy represents a single Cedar policy.
type Policy struct {
	ID        string
	Effect    Effect
	Principal Match // who
	Action    Match // HTTP method or action
	Resource  Match // OpenSearch index pattern
	When      []Condition
	Unless    []Condition
}

// Match is a principal/action/resource matcher.
type Match struct {
	Any     bool   // true = matches anything
	Equals  string // exact match
	Pattern string // glob pattern (e.g., "logs-*")
}

// Condition is a when/unless clause.
type Condition struct {
	Field string // e.g., "principal.scope", "resource.index"
	Op    string // "==", "!=", "in", "contains"
	Value string
}

// Request is an authorization request to evaluate.
type Request struct {
	Principal map[string]string // JWT claims: sub, scope, iss, etc.
	Action    string            // HTTP method: GET, POST, PUT, DELETE
	Resource  map[string]string // index, path, etc.
}

// Decision is the result of policy evaluation.
type Decision struct {
	Allowed bool
	Reason  string
	Policy  string // ID of the deciding policy
}

// Engine evaluates Cedar policies.
type Engine struct {
	mu       sync.RWMutex
	policies []Policy
}

// NewEngine creates a policy engine with the given policies.
func NewEngine(policies []Policy) *Engine {
	return &Engine{policies: policies}
}

// Evaluate runs all policies against a request. Forbid-overrides: any forbid wins.
func (e *Engine) Evaluate(req Request) Decision {
	var permitMatch *Policy

	for i := range e.policies {
		p := &e.policies[i]
		if !matchesTarget(p.Principal, req.Principal["sub"]) {
			continue
		}
		if !matchesTarget(p.Action, req.Action) {
			continue
		}
		if !matchesTarget(p.Resource, req.Resource["index"]) {
			continue
		}
		if !evalConditions(p.When, req) {
			continue
		}
		if len(p.Unless) > 0 && evalConditions(p.Unless, req) {
			continue // unless matched = skip this policy
		}

		if p.Effect == Forbid {
			return Decision{Allowed: false, Reason: "denied by policy", Policy: p.ID}
		}
		if permitMatch == nil {
			permitMatch = p
		}
	}

	if permitMatch != nil {
		return Decision{Allowed: true, Reason: "permitted", Policy: permitMatch.ID}
	}
	return Decision{Allowed: false, Reason: "no matching permit policy"}
}

// Policies returns a copy of all policies.
func (e *Engine) Policies() []Policy {
	out := make([]Policy, len(e.policies))
	copy(out, e.policies)
	return out
}

// AddPolicy appends a policy.
func (e *Engine) AddPolicy(p Policy) {
	e.policies = append(e.policies, p)
}

// RemovePolicy removes a policy by ID. Returns true if found.
func (e *Engine) RemovePolicy(id string) bool {
	for i, p := range e.policies {
		if p.ID == id {
			e.policies = append(e.policies[:i], e.policies[i+1:]...)
			return true
		}
	}
	return false
}

func matchesTarget(m Match, value string) bool {
	if m.Any {
		return true
	}
	if m.Equals != "" {
		return m.Equals == value
	}
	if m.Pattern != "" {
		return globMatch(m.Pattern, value)
	}
	return false
}

func evalConditions(conds []Condition, req Request) bool {
	for _, c := range conds {
		val := resolveField(c.Field, req)
		switch c.Op {
		case "==":
			if val != c.Value {
				return false
			}
		case "!=":
			if val == c.Value {
				return false
			}
		case "contains":
			if !strings.Contains(val, c.Value) {
				return false
			}
		case "in":
			// Check if val is in comma-separated list
			for _, item := range strings.Split(c.Value, ",") {
				if strings.TrimSpace(item) == val {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return true
}

func resolveField(field string, req Request) string {
	parts := strings.SplitN(field, ".", 2)
	if len(parts) == 1 {
		// Unqualified field — check action, principal, resource
		if field == "action" {
			return req.Action
		}
		if v, ok := req.Principal[field]; ok {
			return v
		}
		if v, ok := req.Resource[field]; ok {
			return v
		}
		return ""
	}
	switch parts[0] {
	case "principal":
		return req.Principal[parts[1]]
	case "resource":
		return req.Resource[parts[1]]
	case "action":
		return req.Action
	}
	return ""
}

// globMatch supports simple glob patterns: * matches any sequence.
func globMatch(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == value
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		return strings.HasPrefix(value, parts[0]) && strings.HasSuffix(value, parts[1])
	}
	return false
}

// ParsePolicy parses a simplified Cedar policy string.
// Format: effect(principal, action, resource) [when { conditions }] [unless { conditions }];
func ParsePolicy(id, text string) (Policy, error) {
	text = strings.TrimSpace(text)
	p := Policy{ID: id}

	// Parse effect
	if strings.HasPrefix(text, "permit") {
		p.Effect = Permit
		text = strings.TrimPrefix(text, "permit")
	} else if strings.HasPrefix(text, "forbid") {
		p.Effect = Forbid
		text = strings.TrimPrefix(text, "forbid")
	} else {
		return p, fmt.Errorf("policy must start with permit or forbid")
	}

	// Parse (principal, action, resource)
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "(") {
		return p, fmt.Errorf("expected ( after effect")
	}
	end := strings.Index(text, ")")
	if end < 0 {
		return p, fmt.Errorf("missing )")
	}
	targets := strings.SplitN(text[1:end], ",", 3)
	if len(targets) != 3 {
		return p, fmt.Errorf("expected 3 targets: principal, action, resource")
	}
	p.Principal = parseMatch(strings.TrimSpace(targets[0]))
	p.Action = parseMatch(strings.TrimSpace(targets[1]))
	p.Resource = parseMatch(strings.TrimSpace(targets[2]))

	text = strings.TrimSpace(text[end+1:])

	// Parse when/unless blocks
	for len(text) > 0 {
		text = strings.TrimSpace(text)
		if text == ";" || text == "" {
			break
		}
		if strings.HasPrefix(text, "when") {
			conds, rest, err := parseConditionBlock(strings.TrimPrefix(text, "when"))
			if err != nil {
				return p, fmt.Errorf("when: %w", err)
			}
			p.When = conds
			text = rest
		} else if strings.HasPrefix(text, "unless") {
			conds, rest, err := parseConditionBlock(strings.TrimPrefix(text, "unless"))
			if err != nil {
				return p, fmt.Errorf("unless: %w", err)
			}
			p.Unless = conds
			text = rest
		} else {
			break
		}
	}

	return p, nil
}

func parseMatch(s string) Match {
	if s == "*" {
		return Match{Any: true}
	}
	if strings.Contains(s, "*") {
		return Match{Pattern: s}
	}
	return Match{Equals: s}
}

func parseConditionBlock(text string) ([]Condition, string, error) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "{") {
		return nil, text, fmt.Errorf("expected {")
	}
	end := strings.Index(text, "}")
	if end < 0 {
		return nil, text, fmt.Errorf("missing }")
	}
	body := text[1:end]
	rest := strings.TrimSpace(text[end+1:])

	var conds []Condition
	for _, part := range strings.Split(body, "&&") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		c, err := parseCondition(part)
		if err != nil {
			return nil, rest, err
		}
		conds = append(conds, c)
	}
	return conds, rest, nil
}

func parseCondition(s string) (Condition, error) {
	for _, op := range []string{"!=", "==", " contains ", " in "} {
		if idx := strings.Index(s, op); idx >= 0 {
			return Condition{
				Field: strings.TrimSpace(s[:idx]),
				Op:    strings.TrimSpace(op),
				Value: strings.Trim(strings.TrimSpace(s[idx+len(op):]), "\""),
			}, nil
		}
	}
	return Condition{}, fmt.Errorf("unsupported condition: %s", s)
}
