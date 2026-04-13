package cedar

import (
	"testing"
)

func TestEdge_EngineGlobMatch(t *testing.T) {
	e := NewEngine([]Policy{{
		ID: "allow-logs", Effect: Permit,
		Principal: Match{Any: true},
		Action:    Match{Any: true},
		Resource:  Match{Pattern: "logs-*"},
	}})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "u"},
		Action:    "read",
		Resource:  map[string]string{"index": "logs-2026"},
	})
	if !d.Allowed {
		t.Error("glob logs-* should match logs-2026")
	}
}

func TestEdge_EngineGlobNoMatch(t *testing.T) {
	e := NewEngine([]Policy{{
		ID: "allow-logs", Effect: Permit,
		Principal: Match{Any: true},
		Action:    Match{Any: true},
		Resource:  Match{Pattern: "logs-*"},
	}})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "u"},
		Action:    "read",
		Resource:  map[string]string{"index": "metrics-2026"},
	})
	if d.Allowed {
		t.Error("glob logs-* should not match metrics-2026")
	}
}

func TestEdge_EngineExactMatch(t *testing.T) {
	e := NewEngine([]Policy{{
		ID: "allow-read", Effect: Permit,
		Principal: Match{Any: true},
		Action:    Match{Equals: "read"},
		Resource:  Match{Any: true},
	}})
	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "u"},
		Action:    "write",
		Resource:  map[string]string{"index": "x"},
	})
	if d.Allowed {
		t.Error("exact match 'read' should not match 'write'")
	}
}
