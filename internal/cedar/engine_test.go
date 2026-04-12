package cedar

import "testing"

func TestPermitAll(t *testing.T) {
	e := NewEngine([]Policy{{
		ID: "allow-all", Effect: Permit,
		Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true},
	}})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "agent-1"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-2025"},
	})
	if !d.Allowed {
		t.Fatalf("expected allowed, got denied: %s", d.Reason)
	}
}

func TestForbidOverrides(t *testing.T) {
	e := NewEngine([]Policy{
		{ID: "allow-all", Effect: Permit,
			Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
		{ID: "deny-security", Effect: Forbid,
			Principal: Match{Any: true}, Action: Match{Any: true},
			Resource: Match{Equals: ".opendistro_security"}},
	})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "agent-1"},
		Action:    "GET",
		Resource:  map[string]string{"index": ".opendistro_security"},
	})
	if d.Allowed {
		t.Fatal("expected denied for security index")
	}
	if d.Policy != "deny-security" {
		t.Fatalf("expected deny-security policy, got %s", d.Policy)
	}
}

func TestGlobPattern(t *testing.T) {
	e := NewEngine([]Policy{{
		ID: "read-logs", Effect: Permit,
		Principal: Match{Any: true}, Action: Match{Equals: "GET"},
		Resource: Match{Pattern: "logs-*"},
	}})
	// Should match
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "agent"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-2025-04"},
	})
	if !d.Allowed {
		t.Fatal("expected allowed for logs-2025-04")
	}
	// Should not match
	d = e.Evaluate(Request{
		Principal: map[string]string{"sub": "agent"},
		Action:    "GET",
		Resource:  map[string]string{"index": "metrics-2025"},
	})
	if d.Allowed {
		t.Fatal("expected denied for metrics-2025")
	}
}

func TestWhenCondition(t *testing.T) {
	e := NewEngine([]Policy{{
		ID: "scope-check", Effect: Permit,
		Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true},
		When: []Condition{{Field: "principal.scope", Op: "contains", Value: "read:logs"}},
	}})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "agent", "scope": "read:logs-* write:dashboards"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-2025"},
	})
	if !d.Allowed {
		t.Fatal("expected allowed with matching scope")
	}
	d = e.Evaluate(Request{
		Principal: map[string]string{"sub": "agent", "scope": "write:dashboards"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-2025"},
	})
	if d.Allowed {
		t.Fatal("expected denied without matching scope")
	}
}

func TestUnlessCondition(t *testing.T) {
	e := NewEngine([]Policy{{
		ID: "allow-unless-admin", Effect: Permit,
		Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true},
		Unless: []Condition{{Field: "resource.index", Op: "==", Value: ".opendistro_security"}},
	}})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "agent"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-2025"},
	})
	if !d.Allowed {
		t.Fatal("expected allowed for normal index")
	}
	d = e.Evaluate(Request{
		Principal: map[string]string{"sub": "agent"},
		Action:    "GET",
		Resource:  map[string]string{"index": ".opendistro_security"},
	})
	if d.Allowed {
		t.Fatal("expected denied for security index")
	}
}

func TestNoMatchingPolicy(t *testing.T) {
	e := NewEngine([]Policy{{
		ID: "read-only", Effect: Permit,
		Principal: Match{Equals: "admin"}, Action: Match{Any: true}, Resource: Match{Any: true},
	}})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "agent"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs"},
	})
	if d.Allowed {
		t.Fatal("expected denied — no matching policy")
	}
}

func TestParsePolicy(t *testing.T) {
	p, err := ParsePolicy("test", `permit(*, GET, logs-*) when { principal.scope contains "read:logs" };`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if p.Effect != Permit {
		t.Fatal("expected permit")
	}
	if !p.Principal.Any {
		t.Fatal("expected principal=*")
	}
	if p.Action.Equals != "GET" {
		t.Fatalf("expected action=GET, got %s", p.Action.Equals)
	}
	if p.Resource.Pattern != "logs-*" {
		t.Fatalf("expected resource=logs-*, got %s", p.Resource.Pattern)
	}
	if len(p.When) != 1 || p.When[0].Value != "read:logs" {
		t.Fatalf("expected when condition, got %+v", p.When)
	}
}

func TestParseForbidWithUnless(t *testing.T) {
	p, err := ParsePolicy("deny", `forbid(*, *, .opendistro_security) unless { principal.role == "admin" };`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if p.Effect != Forbid {
		t.Fatal("expected forbid")
	}
	if len(p.Unless) != 1 || p.Unless[0].Field != "principal.role" {
		t.Fatalf("expected unless condition, got %+v", p.Unless)
	}
}

func TestParseEmptyPolicy(t *testing.T) {
	_, err := ParsePolicy("empty", "")
	if err == nil {
		t.Fatal("expected error for empty policy")
	}
}

func TestParseMissingParens(t *testing.T) {
	_, err := ParsePolicy("bad", "permit *, *, *;")
	if err == nil {
		t.Fatal("expected error for missing parens")
	}
}

func TestParseTwoTargets(t *testing.T) {
	_, err := ParsePolicy("bad", "permit(*, *);")
	if err == nil {
		t.Fatal("expected error for 2 targets instead of 3")
	}
}

func TestParseBadEffect(t *testing.T) {
	_, err := ParsePolicy("bad", "allow(*, *, *);")
	if err == nil {
		t.Fatal("expected error for invalid effect")
	}
}

func TestParseBadConditionOp(t *testing.T) {
	_, err := ParsePolicy("bad", `permit(*, *, *) when { principal.sub > "admin" };`)
	if err == nil {
		t.Fatal("expected error for unsupported operator >")
	}
}

func TestEvaluateEmptyEngine(t *testing.T) {
	e := NewEngine(nil)
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "x"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs"},
	})
	if d.Allowed {
		t.Fatal("expected deny with no policies")
	}
}

func TestGlobMatchEdgeCases(t *testing.T) {
	cases := []struct {
		pattern, value string
		want           bool
	}{
		{"*", "anything", true},
		{"*", "", true},
		{"logs-*", "logs-", true},
		{"logs-*", "logs-2025", true},
		{"logs-*", "metrics-2025", false},
		{"exact", "exact", true},
		{"exact", "other", false},
		{"*.json", "data.json", true},
		{"*.json", "data.csv", false},
	}
	for _, c := range cases {
		got := globMatch(c.pattern, c.value)
		if got != c.want {
			t.Errorf("globMatch(%q, %q) = %v, want %v", c.pattern, c.value, got, c.want)
		}
	}
}
