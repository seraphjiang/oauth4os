package cedar

import "testing"

// Mutation tests for Cedar engine.

// Mutation: What if forbid didn't override permit?
func TestMutation_ForbidIgnored(t *testing.T) {
	e := NewEngine([]Policy{
		{ID: "allow", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
		{ID: "deny", Effect: Forbid, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Equals: ".opendistro_security"}},
	})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "user"},
		Action:    "GET",
		Resource:  map[string]string{"index": ".opendistro_security"},
	})
	if d.Allowed {
		t.Error("MUTATION SURVIVED: forbid doesn't override permit")
	}
}

// Mutation: What if empty policy set allowed by default?
func TestMutation_EmptyPoliciesAllow(t *testing.T) {
	e := NewEngine(nil)
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "user"},
		Action:    "DELETE",
		Resource:  map[string]string{"index": "production"},
	})
	if d.Allowed {
		t.Error("MUTATION SURVIVED: empty policy set allows requests")
	}
}

// Mutation: What if glob * matched nothing?
func TestMutation_GlobStarBroken(t *testing.T) {
	e := NewEngine([]Policy{
		{ID: "p", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Pattern: "logs-*"}},
	})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "u"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-2026"},
	})
	if !d.Allowed {
		t.Error("MUTATION SURVIVED: glob pattern logs-* doesn't match logs-2026")
	}
}

// Mutation: What if when condition was ignored?
func TestMutation_WhenConditionIgnored(t *testing.T) {
	p, _ := ParsePolicy("p", `permit(*, *, *) when { principal.scope == "admin" };`)
	e := NewEngine([]Policy{p})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "u", "scope": "readonly"},
		Action:    "DELETE",
		Resource:  map[string]string{"index": "x"},
	})
	if d.Allowed {
		t.Error("MUTATION SURVIVED: when condition not evaluated")
	}
}

// Mutation: What if unless condition was ignored?
func TestMutation_UnlessConditionIgnored(t *testing.T) {
	p, _ := ParsePolicy("p", `permit(*, *, *) unless { principal.scope == "readonly" };`)
	e := NewEngine([]Policy{p})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "u", "scope": "readonly"},
		Action:    "GET",
		Resource:  map[string]string{"index": "x"},
	})
	if d.Allowed {
		t.Error("MUTATION SURVIVED: unless condition not evaluated")
	}
}

// Mutation: What if exact match used Contains instead of ==?
func TestMutation_ExactMatchLoose(t *testing.T) {
	e := NewEngine([]Policy{
		{ID: "allow", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
		{ID: "deny", Effect: Forbid, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Equals: "secret"}},
	})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "u"},
		Action:    "GET",
		Resource:  map[string]string{"index": "not-secret-at-all"},
	})
	// "not-secret-at-all" contains "secret" but should NOT match exact "secret"
	if !d.Allowed {
		t.Error("MUTATION SURVIVED: exact match uses substring instead of equality")
	}
}


