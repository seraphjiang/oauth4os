package cedar

import "testing"

// --- ParsePolicy edge cases (#19) ---

func TestParseEmpty(t *testing.T) {
	_, err := ParsePolicy("e", "")
	if err == nil {
		t.Fatal("expected error for empty policy")
	}
}

func TestParseNoEffect(t *testing.T) {
	_, err := ParsePolicy("e", "allow(*, *, *)")
	if err == nil {
		t.Fatal("expected error for unknown effect")
	}
}

func TestParseMissingParen(t *testing.T) {
	_, err := ParsePolicy("e", "permit *, *, *)")
	if err == nil {
		t.Fatal("expected error for missing (")
	}
}

func TestParseMissingCloseParen(t *testing.T) {
	_, err := ParsePolicy("e", "permit(*, *, *")
	if err == nil {
		t.Fatal("expected error for missing )")
	}
}

func TestParseTooFewTargets(t *testing.T) {
	_, err := ParsePolicy("e", "permit(*, *)")
	if err == nil {
		t.Fatal("expected error for 2 targets")
	}
}

func TestParseOneTarget(t *testing.T) {
	_, err := ParsePolicy("e", "permit(*)")
	if err == nil {
		t.Fatal("expected error for 1 target")
	}
}

func TestParseWhenMissingBrace(t *testing.T) {
	_, err := ParsePolicy("e", `permit(*, *, *) when principal.scope == "x"`)
	if err == nil {
		t.Fatal("expected error for when without {")
	}
}

func TestParseWhenMissingCloseBrace(t *testing.T) {
	_, err := ParsePolicy("e", `permit(*, *, *) when { principal.scope == "x"`)
	if err == nil {
		t.Fatal("expected error for when without }")
	}
}

func TestParseUnlessMissingBrace(t *testing.T) {
	_, err := ParsePolicy("e", `permit(*, *, *) unless resource.index == "x"`)
	if err == nil {
		t.Fatal("expected error for unless without {")
	}
}

func TestParseEmptyWhenBlock(t *testing.T) {
	p, err := ParsePolicy("e", `permit(*, *, *) when { }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.When) != 0 {
		t.Fatalf("expected 0 when conditions, got %d", len(p.When))
	}
}

func TestParseEmptyUnlessBlock(t *testing.T) {
	p, err := ParsePolicy("e", `permit(*, *, *) unless { }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Unless) != 0 {
		t.Fatalf("expected 0 unless conditions, got %d", len(p.Unless))
	}
}

func TestParseBadConditionOp(t *testing.T) {
	_, err := ParsePolicy("e", `permit(*, *, *) when { principal.scope > "x" }`)
	if err == nil {
		t.Fatal("expected error for unsupported operator >")
	}
}

func TestParseMultipleConditions(t *testing.T) {
	p, err := ParsePolicy("e", `permit(*, *, *) when { principal.scope == "admin" && principal.iss == "keycloak" }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.When) != 2 {
		t.Fatalf("expected 2 when conditions, got %d", len(p.When))
	}
}

func TestParseWhenAndUnless(t *testing.T) {
	p, err := ParsePolicy("e", `permit(*, GET, logs-*) when { principal.scope == "read" } unless { resource.index == ".internal" }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.When) != 1 || len(p.Unless) != 1 {
		t.Fatalf("expected 1 when + 1 unless, got %d + %d", len(p.When), len(p.Unless))
	}
}

func TestParseSemicolon(t *testing.T) {
	p, err := ParsePolicy("e", `permit(*, *, *);`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Effect != Permit {
		t.Fatal("expected permit")
	}
}

func TestParseWhitespace(t *testing.T) {
	p, err := ParsePolicy("e", `   permit(  *  ,  *  ,  *  )   ;   `)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.Principal.Any || !p.Action.Any || !p.Resource.Any {
		t.Fatal("expected all wildcards")
	}
}

// --- Evaluate edge cases ---

func TestEvaluateEmptyPolicies(t *testing.T) {
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

func TestEvaluateNilMaps(t *testing.T) {
	e := NewEngine([]Policy{{
		ID: "p", Effect: Permit,
		Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true},
	}})
	d := e.Evaluate(Request{})
	if !d.Allowed {
		t.Fatal("expected permit with nil maps and Any matchers")
	}
}

func TestEvaluateEmptyStrings(t *testing.T) {
	e := NewEngine([]Policy{{
		ID: "p", Effect: Permit,
		Principal: Match{Equals: ""}, Action: Match{Equals: ""}, Resource: Match{Equals: ""},
	}})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": ""},
		Action:    "",
		Resource:  map[string]string{"index": ""},
	})
	if !d.Allowed {
		t.Fatal("expected permit with empty string equals")
	}
}

func TestEvaluateConditionUnknownOp(t *testing.T) {
	e := NewEngine([]Policy{{
		ID: "p", Effect: Permit,
		Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true},
		When: []Condition{{Field: "principal.sub", Op: ">=", Value: "x"}},
	}})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "x"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs"},
	})
	if d.Allowed {
		t.Fatal("expected deny — unknown op should fail condition")
	}
}

func TestEvaluateFieldNoPrefix(t *testing.T) {
	e := NewEngine([]Policy{{
		ID: "p", Effect: Permit,
		Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true},
		When: []Condition{{Field: "noDot", Op: "==", Value: "x"}},
	}})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "x"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs"},
	})
	if d.Allowed {
		t.Fatal("expected deny — field without dot resolves to empty")
	}
}

func TestGlobMatchEmpty(t *testing.T) {
	if globMatch("", "") {
		// empty pattern with empty value — no wildcard, exact match
	}
	if globMatch("*", "") != true {
		t.Fatal("* should match empty string")
	}
}

func TestGlobMatchNoWildcard(t *testing.T) {
	if !globMatch("exact", "exact") {
		t.Fatal("exact match should work")
	}
	if globMatch("exact", "other") {
		t.Fatal("non-matching exact should fail")
	}
}

func TestMatchEmptyPattern(t *testing.T) {
	m := Match{}
	if matchesTarget(m, "anything") {
		t.Fatal("empty match should not match")
	}
}
