package cedar

import (
	"testing"
	"testing/quick"
)

// Property: Forbid always overrides permit for the same resource.
func TestProperty_ForbidOverridesPermit(t *testing.T) {
	f := func(index string) bool {
		if index == "" {
			return true
		}
		e := NewEngine([]Policy{
			{ID: "allow", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
			{ID: "deny", Effect: Forbid, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Equals: index}},
		})
		d := e.Evaluate(Request{
			Principal: map[string]string{"sub": "user"},
			Action:    "GET",
			Resource:  map[string]string{"index": index},
		})
		// INVARIANT: explicit forbid must deny even with permit-all
		return !d.Allowed
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 200}); err != nil {
		t.Error(err)
	}
}

// Property: No policies = always deny.
func TestProperty_NoPoliciesDeny(t *testing.T) {
	f := func(sub, action, index string) bool {
		e := NewEngine(nil)
		d := e.Evaluate(Request{
			Principal: map[string]string{"sub": sub},
			Action:    action,
			Resource:  map[string]string{"index": index},
		})
		return !d.Allowed
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 200}); err != nil {
		t.Error(err)
	}
}

// Property: ParsePolicy never panics on any input.
func TestProperty_ParseNeverPanics(t *testing.T) {
	f := func(input string) bool {
		ParsePolicy("test", input) // must not panic
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 500}); err != nil {
		t.Error(err)
	}
}

// Property: Permit-all with no forbid always allows.
func TestProperty_PermitAllAlwaysAllows(t *testing.T) {
	f := func(sub, action, index string) bool {
		e := NewEngine([]Policy{
			{ID: "all", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
		})
		d := e.Evaluate(Request{
			Principal: map[string]string{"sub": sub},
			Action:    action,
			Resource:  map[string]string{"index": index},
		})
		return d.Allowed
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 200}); err != nil {
		t.Error(err)
	}
}
