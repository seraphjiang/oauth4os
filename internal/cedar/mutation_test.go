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



// Mutation: remove AddPolicy → new policy must affect evaluation
func TestMutation_AddPolicyAffectsEval(t *testing.T) {
	e := NewEngine(nil)
	req := Request{Principal: map[string]string{"sub": "alice"}, Action: "read", Resource: map[string]string{"index": "logs"}}
	d1 := e.Evaluate(req)
	e.AddPolicy(Policy{ID: "deny-alice", Effect: Forbid, Principal: Match{Equals: "alice"}, Action: Match{Equals: "read"}, Resource: Match{Pattern: "logs"}})
	d2 := e.Evaluate(req)
	if d1 == d2 {
		t.Error("AddPolicy must change evaluation result")
	}
}

// Mutation: remove RemovePolicy → removed policy must stop affecting evaluation
func TestMutation_RemovePolicyRestores(t *testing.T) {
	e := NewEngine([]Policy{
		{ID: "deny-all", Effect: Forbid, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
	})
	req := Request{Principal: map[string]string{"sub": "bob"}, Action: "read", Resource: map[string]string{"index": "logs"}}
	d1 := e.Evaluate(req)
	e.RemovePolicy("deny-all")
	d2 := e.Evaluate(req)
	if d1 == d2 {
		t.Error("RemovePolicy must change evaluation result")
	}
}

// Mutation: remove tenant isolation → tenant policies must not leak
func TestMutation_TenantIsolation(t *testing.T) {
	te := NewTenantEngine(nil)
	te.AddTenant("issuer-a", []Policy{
		{ID: "a-deny", Effect: Forbid, Principal: Match{Any: true}, Action: Match{Equals: "delete"}, Resource: Match{Any: true}},
	})
	te.AddTenant("issuer-b", nil)
	req := Request{Principal: map[string]string{"sub": "x"}, Action: "delete", Resource: map[string]string{"index": "logs"}}
	dA := te.Evaluate("issuer-a", req)
	dB := te.Evaluate("issuer-b", req)
	if dA == dB {
		t.Error("tenant policies must be isolated — issuer-a deny should not affect issuer-b")
	}
}

// Mutation: remove ListPolicies → must return global policies
func TestMutation_ListPolicies(t *testing.T) {
	te := NewTenantEngine([]Policy{
		{ID: "p1", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
	})
	policies := te.ListPolicies()
	if len(policies) == 0 {
		t.Error("ListPolicies must return global policies")
	}
}

// Mutation: remove AddGlobalPolicy → global policy must affect all tenants
func TestMutation_AddGlobalPolicy(t *testing.T) {
	te := NewTenantEngine(nil)
	te.AddTenant("issuer-a", nil)
	te.AddGlobalPolicy(Policy{
		ID: "global-deny", Effect: Forbid,
		Principal: Match{Any: true}, Action: Match{Equals: "delete"}, Resource: Match{Any: true},
	})
	req := Request{
		Principal: map[string]string{"sub": "user"},
		Action:    "delete",
		Resource:  map[string]string{"index": "logs"},
	}
	d := te.Evaluate("issuer-a", req)
	if d.Allowed {
		t.Error("global deny policy must affect all tenants")
	}
}
