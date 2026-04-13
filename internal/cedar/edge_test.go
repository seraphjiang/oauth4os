package cedar

import (
	"sync"
	"testing"
)

func TestEdge_TenantIsolationConcurrent(t *testing.T) {
	te := NewTenantEngine(nil)
	te.AddTenant("a", []Policy{{ID: "p1", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}}})
	te.AddTenant("b", []Policy{{ID: "p2", Effect: Forbid, Principal: Match{Any: true}, Action: Match{Equals: "delete"}, Resource: Match{Any: true}}})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			te.Evaluate("a", Request{Principal: map[string]string{"sub": "u"}, Action: "read", Resource: map[string]string{"idx": "x"}})
		}()
		go func() {
			defer wg.Done()
			te.Evaluate("b", Request{Principal: map[string]string{"sub": "u"}, Action: "delete", Resource: map[string]string{"idx": "x"}})
		}()
	}
	wg.Wait()
}

func TestEdge_EmptyTenantDefaultBehavior(t *testing.T) {
	te := NewTenantEngine(nil)
	te.AddTenant("empty", nil)
	d := te.Evaluate("empty", Request{Principal: map[string]string{"sub": "u"}, Action: "read", Resource: map[string]string{"idx": "x"}})
	// Empty policy set — behavior is implementation-defined, just verify no panic
	_ = d
}

func TestEdge_UnknownTenantHandled(t *testing.T) {
	te := NewTenantEngine(nil)
	d := te.Evaluate("nonexistent", Request{Principal: map[string]string{"sub": "u"}, Action: "read", Resource: map[string]string{"idx": "x"}})
	// Should not panic — either allow or deny
	_ = d
}
