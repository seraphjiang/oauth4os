package cedar

import "testing"

func BenchmarkEvaluatePermit(b *testing.B) {
	e := NewEngine([]Policy{
		{ID: "p1", Effect: "permit", Principal: Match{Any: true}, Action: Match{Equals: "GET"}, Resource: Match{Pattern: "logs-*"}},
	})
	req := Request{
		Principal: map[string]string{"sub": "user-1"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-app"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Evaluate(req)
	}
}

func BenchmarkEvaluateForbidOverride(b *testing.B) {
	e := NewEngine([]Policy{
		{ID: "p1", Effect: "permit", Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
		{ID: "p2", Effect: "forbid", Principal: Match{Equals: "attacker"}, Action: Match{Any: true}, Resource: Match{Any: true}},
	})
	req := Request{
		Principal: map[string]string{"sub": "attacker"},
		Action:    "DELETE",
		Resource:  map[string]string{"index": "prod-data"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Evaluate(req)
	}
}

func BenchmarkEvaluate10Policies(b *testing.B) {
	policies := make([]Policy, 10)
	for i := range policies {
		policies[i] = Policy{ID: string(rune('A' + i)), Effect: "permit", Principal: Match{Any: true}, Action: Match{Equals: "GET"}, Resource: Match{Pattern: "logs-*"}}
	}
	e := NewEngine(policies)
	req := Request{
		Principal: map[string]string{"sub": "user"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-app"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Evaluate(req)
	}
}
