package cedar

import (
	"testing"
)

func TestEdge_EnginePermitAll(t *testing.T) {
	e := NewEngine([]Policy{{ID: "allow-all", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}}})
	d := e.Evaluate(Request{Principal: map[string]string{"sub": "u"}, Action: "read", Resource: map[string]string{"idx": "x"}})
	if !d.Allowed {
		t.Error("permit-all should allow")
	}
}

func TestEdge_EngineForbidOverridesPermit(t *testing.T) {
	e := NewEngine([]Policy{
		{ID: "allow", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
		{ID: "deny-delete", Effect: Forbid, Principal: Match{Any: true}, Action: Match{Equals: "delete"}, Resource: Match{Any: true}},
	})
	d := e.Evaluate(Request{Principal: map[string]string{"sub": "u"}, Action: "delete", Resource: map[string]string{"idx": "x"}})
	if d.Allowed {
		t.Error("forbid should override permit")
	}
}

func TestEdge_EngineNoPoliciesDenies(t *testing.T) {
	e := NewEngine(nil)
	d := e.Evaluate(Request{Principal: map[string]string{"sub": "u"}, Action: "read", Resource: map[string]string{"idx": "x"}})
	if d.Allowed {
		t.Error("no policies should deny by default")
	}
}
