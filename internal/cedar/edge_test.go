package cedar

import "testing"

// Edge case tests for Cedar policy parsing and evaluation.

func TestParsePolicy_EmptyString(t *testing.T) {
	_, err := ParsePolicy("e1", "")
	if err == nil {
		t.Error("empty string should fail")
	}
}

func TestParsePolicy_WhitespaceOnly(t *testing.T) {
	_, err := ParsePolicy("e2", "   \n\t  ")
	if err == nil {
		t.Error("whitespace-only should fail")
	}
}

func TestParsePolicy_NoParens(t *testing.T) {
	_, err := ParsePolicy("e3", "permit *,*,*;")
	if err == nil {
		t.Error("missing parens should fail")
	}
}

func TestParsePolicy_UnclosedParen(t *testing.T) {
	_, err := ParsePolicy("e4", "permit(*, *, *")
	if err == nil {
		t.Error("unclosed paren should fail")
	}
}

func TestParsePolicy_TwoTargets(t *testing.T) {
	_, err := ParsePolicy("e5", "permit(*, *);")
	if err == nil {
		t.Error("2 targets should fail")
	}
}

func TestParsePolicy_EmptyTargets(t *testing.T) {
	p, err := ParsePolicy("e6", "permit(, , );")
	if err != nil {
		t.Fatalf("empty targets should parse: %v", err)
	}
	// Empty targets match nothing (not Any)
	if p.Principal.Any || p.Action.Any || p.Resource.Any {
		t.Error("empty targets should not be Any")
	}
}

func TestParsePolicy_WhenEmptyBlock(t *testing.T) {
	p, err := ParsePolicy("e7", "permit(*, *, *) when { };")
	if err != nil {
		t.Fatalf("empty when block should parse: %v", err)
	}
	if len(p.When) != 0 {
		t.Errorf("expected 0 conditions, got %d", len(p.When))
	}
}

func TestParsePolicy_WhenUnclosedBrace(t *testing.T) {
	_, err := ParsePolicy("e8", "permit(*, *, *) when { scope == \"admin\"")
	if err == nil {
		t.Error("unclosed brace should fail")
	}
}

func TestParsePolicy_WhenBadCondition(t *testing.T) {
	_, err := ParsePolicy("e9", `permit(*, *, *) when { scope };`)
	if err == nil {
		t.Error("condition without operator should fail")
	}
}

func TestParsePolicy_UnlessAndWhen(t *testing.T) {
	p, err := ParsePolicy("e10", `permit(*, *, *) when { scope == "admin" } unless { index == ".internal" };`)
	if err != nil {
		t.Fatalf("when+unless should parse: %v", err)
	}
	if len(p.When) != 1 || len(p.Unless) != 1 {
		t.Errorf("expected 1 when + 1 unless, got %d + %d", len(p.When), len(p.Unless))
	}
}

func TestParsePolicy_MultipleConditions(t *testing.T) {
	p, err := ParsePolicy("e11", `forbid(*, *, *) when { scope == "read" && index == "secret" };`)
	if err != nil {
		t.Fatalf("multiple conditions should parse: %v", err)
	}
	if len(p.When) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(p.When))
	}
}

func TestParsePolicy_NotEqualsOperator(t *testing.T) {
	p, err := ParsePolicy("e12", `permit(*, *, *) when { scope != "readonly" };`)
	if err != nil {
		t.Fatalf("!= should parse: %v", err)
	}
	if p.When[0].Op != "!=" {
		t.Errorf("expected !=, got %s", p.When[0].Op)
	}
}

func TestParsePolicy_ContainsOperator(t *testing.T) {
	p, err := ParsePolicy("e13", `permit(*, *, *) when { scope contains "admin" };`)
	if err != nil {
		t.Fatalf("contains should parse: %v", err)
	}
	if p.When[0].Op != "contains" {
		t.Errorf("expected contains, got %s", p.When[0].Op)
	}
}

func TestParsePolicy_GlobResource(t *testing.T) {
	p, err := ParsePolicy("e14", "forbid(*, DELETE, logs-*);")
	if err != nil {
		t.Fatalf("glob resource should parse: %v", err)
	}
	if p.Resource.Pattern != "logs-*" {
		t.Errorf("expected glob pattern, got %+v", p.Resource)
	}
}

func TestParsePolicy_ExactMatch(t *testing.T) {
	p, err := ParsePolicy("e15", "forbid(*, *, .opendistro_security);")
	if err != nil {
		t.Fatalf("exact match should parse: %v", err)
	}
	if p.Resource.Equals != ".opendistro_security" {
		t.Errorf("expected exact match, got %+v", p.Resource)
	}
}

func TestParsePolicy_GarbageAfterSemicolon(t *testing.T) {
	// Parser should stop at semicolon — garbage after is ignored
	p, err := ParsePolicy("e16", "permit(*, *, *); this is garbage")
	if err != nil {
		t.Fatalf("should parse up to semicolon: %v", err)
	}
	if p.Effect != Permit {
		t.Error("expected permit")
	}
}

// Evaluation edge cases

func TestEvaluate_EmptyPolicySet(t *testing.T) {
	e := NewEngine(nil)
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "user"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs"},
	})
	// No policies = default deny
	if d.Allowed {
		t.Error("empty policy set should deny")
	}
}

func TestEvaluate_EmptyRequest(t *testing.T) {
	e := NewEngine([]Policy{
		{ID: "p1", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
	})
	d := e.Evaluate(Request{})
	if !d.Allowed {
		t.Error("permit-all should allow empty request")
	}
}

func TestGlobMatch_EmptyPattern(t *testing.T) {
	if globMatch("", "anything") {
		t.Error("empty pattern should not match non-empty value")
	}
}

func TestGlobMatch_EmptyValue(t *testing.T) {
	if globMatch("logs-*", "") {
		t.Error("non-empty pattern should not match empty value")
	}
}

func TestGlobMatch_BothEmpty(t *testing.T) {
	if !globMatch("", "") {
		t.Error("empty pattern should match empty value")
	}
}

func TestGlobMatch_StarOnly(t *testing.T) {
	if !globMatch("*", "anything-at-all") {
		t.Error("* should match anything")
	}
}

func TestGlobMatch_MiddleStar(t *testing.T) {
	if !globMatch("logs-*-prod", "logs-2026-prod") {
		t.Error("middle glob should match")
	}
	if globMatch("logs-*-prod", "logs-2026-staging") {
		t.Error("middle glob should not match wrong suffix")
	}
}
