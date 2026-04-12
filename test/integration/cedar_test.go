// Cedar policy integration tests — verify policy engine evaluates correctly
// in the context of the proxy.
//
// These test the Cedar engine directly (no Docker needed) since the engine
// is a pure Go library.

package integration

import (
	"testing"

	"github.com/seraphjiang/oauth4os/internal/cedar"
)

func TestCedar_PermitReadLogs(t *testing.T) {
	policies := []cedar.Policy{
		{
			ID: "allow-read-logs", Effect: cedar.Permit,
			Principal: cedar.Match{Any: true},
			Action:    cedar.Match{Equals: "GET"},
			Resource:  cedar.Match{Pattern: "logs-*"},
		},
	}
	engine := cedar.NewEngine(policies)
	decision := engine.Evaluate(cedar.Request{
		Principal: map[string]string{"sub": "agent-1", "scope": "read:logs-*"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-2026.04"},
	})
	if !decision.Allowed {
		t.Fatalf("expected permit, got deny: %s", decision.Reason)
	}
}

func TestCedar_ForbidSecurityIndex(t *testing.T) {
	policies := []cedar.Policy{
		{
			ID: "allow-all", Effect: cedar.Permit,
			Principal: cedar.Match{Any: true},
			Action:    cedar.Match{Any: true},
			Resource:  cedar.Match{Any: true},
		},
		{
			ID: "deny-security", Effect: cedar.Forbid,
			Principal: cedar.Match{Any: true},
			Action:    cedar.Match{Any: true},
			Resource:  cedar.Match{Equals: ".opendistro_security"},
		},
	}
	engine := cedar.NewEngine(policies)

	// Normal index — allowed
	d1 := engine.Evaluate(cedar.Request{
		Principal: map[string]string{"sub": "agent"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-2026"},
	})
	if !d1.Allowed {
		t.Fatal("expected permit for normal index")
	}

	// Security index — denied
	d2 := engine.Evaluate(cedar.Request{
		Principal: map[string]string{"sub": "agent"},
		Action:    "GET",
		Resource:  map[string]string{"index": ".opendistro_security"},
	})
	if d2.Allowed {
		t.Fatal("expected deny for security index")
	}
}

func TestCedar_WhenScopeCondition(t *testing.T) {
	policies := []cedar.Policy{
		{
			ID: "scope-check", Effect: cedar.Permit,
			Principal: cedar.Match{Any: true},
			Action:    cedar.Match{Equals: "GET"},
			Resource:  cedar.Match{Pattern: "logs-*"},
			When: []cedar.Condition{
				{Field: "principal.scope", Op: "==", Value: "read:logs-*"},
			},
		},
	}
	engine := cedar.NewEngine(policies)

	// Matching scope
	d1 := engine.Evaluate(cedar.Request{
		Principal: map[string]string{"scope": "read:logs-*"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-2026"},
	})
	if !d1.Allowed {
		t.Fatal("expected permit with matching scope")
	}

	// Wrong scope
	d2 := engine.Evaluate(cedar.Request{
		Principal: map[string]string{"scope": "write:metrics"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-2026"},
	})
	if d2.Allowed {
		t.Fatal("expected deny with wrong scope")
	}
}

func TestCedar_UnlessCondition(t *testing.T) {
	policies := []cedar.Policy{
		{
			ID: "allow-unless-delete", Effect: cedar.Permit,
			Principal: cedar.Match{Any: true},
			Action:    cedar.Match{Any: true},
			Resource:  cedar.Match{Pattern: "logs-*"},
			Unless: []cedar.Condition{
				{Field: "action", Op: "==", Value: "DELETE"},
			},
		},
	}
	engine := cedar.NewEngine(policies)

	d1 := engine.Evaluate(cedar.Request{
		Principal: map[string]string{"sub": "agent"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-2026"},
	})
	if !d1.Allowed {
		t.Fatal("expected permit for GET")
	}

	d2 := engine.Evaluate(cedar.Request{
		Principal: map[string]string{"sub": "agent"},
		Action:    "DELETE",
		Resource:  map[string]string{"index": "logs-2026"},
	})
	if d2.Allowed {
		t.Fatal("expected deny for DELETE")
	}
}

func TestCedar_MultipleProviders(t *testing.T) {
	// Simulate multi-provider: different issuers get different policies
	policies := []cedar.Policy{
		{
			ID: "keycloak-read", Effect: cedar.Permit,
			Principal: cedar.Match{Any: true},
			Action:    cedar.Match{Equals: "GET"},
			Resource:  cedar.Match{Pattern: "logs-*"},
			When: []cedar.Condition{
				{Field: "principal.iss", Op: "==", Value: "https://keycloak.example.com"},
			},
		},
		{
			ID: "auth0-admin", Effect: cedar.Permit,
			Principal: cedar.Match{Any: true},
			Action:    cedar.Match{Any: true},
			Resource:  cedar.Match{Any: true},
			When: []cedar.Condition{
				{Field: "principal.iss", Op: "==", Value: "https://auth0.example.com"},
			},
		},
	}
	engine := cedar.NewEngine(policies)

	// Keycloak user — read only
	d1 := engine.Evaluate(cedar.Request{
		Principal: map[string]string{"iss": "https://keycloak.example.com"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-2026"},
	})
	if !d1.Allowed {
		t.Fatal("keycloak GET should be allowed")
	}

	d2 := engine.Evaluate(cedar.Request{
		Principal: map[string]string{"iss": "https://keycloak.example.com"},
		Action:    "DELETE",
		Resource:  map[string]string{"index": "logs-2026"},
	})
	if d2.Allowed {
		t.Fatal("keycloak DELETE should be denied")
	}

	// Auth0 user — full access
	d3 := engine.Evaluate(cedar.Request{
		Principal: map[string]string{"iss": "https://auth0.example.com"},
		Action:    "DELETE",
		Resource:  map[string]string{"index": "logs-2026"},
	})
	if !d3.Allowed {
		t.Fatal("auth0 DELETE should be allowed")
	}
}
