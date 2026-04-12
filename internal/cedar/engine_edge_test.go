package cedar

import (
	"fmt"
	"sync"
	"testing"
)

func any() Match    { return Match{Any: true} }
func eq(s string) Match { return Match{Equals: s} }
func glob(s string) Match { return Match{Pattern: s} }

func TestConcurrentEvaluate(t *testing.T) {
	e := NewEngine([]Policy{
		{ID: "p1", Effect: "permit", Principal: any(), Action: eq("GET"), Resource: glob("logs-*")},
		{ID: "p2", Effect: "forbid", Principal: eq("attacker"), Action: any(), Resource: any()},
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			d := e.Evaluate(Request{
				Principal: map[string]string{"sub": fmt.Sprintf("user-%d", n)},
				Action:    "GET",
				Resource:  map[string]string{"index": "logs-app"},
			})
			if !d.Allowed {
				t.Errorf("user-%d should be allowed", n)
			}
		}(i)
	}
	wg.Wait()
}

func TestConcurrentAddAndEvaluate(t *testing.T) {
	e := NewEngine([]Policy{
		{ID: "base", Effect: "permit", Principal: any(), Action: any(), Resource: any()},
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			e.AddPolicy(Policy{
				ID: fmt.Sprintf("dyn-%d", n), Effect: "permit",
				Principal: eq(fmt.Sprintf("svc-%d", n)), Action: eq("GET"), Resource: glob("logs-*"),
			})
		}(i)
	}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.Evaluate(Request{
				Principal: map[string]string{"sub": "user"},
				Action:    "GET",
				Resource:  map[string]string{"index": "logs-app"},
			})
		}()
	}
	wg.Wait()
}

func TestForbidOverridesMultiplePermits(t *testing.T) {
	e := NewEngine([]Policy{
		{ID: "p1", Effect: "permit", Principal: any(), Action: any(), Resource: any()},
		{ID: "p2", Effect: "permit", Principal: eq("admin"), Action: any(), Resource: any()},
		{ID: "p3", Effect: "forbid", Principal: eq("admin"), Action: eq("DELETE"), Resource: glob("prod-*")},
	})

	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "admin"},
		Action:    "DELETE",
		Resource:  map[string]string{"index": "prod-logs"},
	})
	if d.Allowed {
		t.Fatal("forbid should override multiple permits")
	}
}

func TestRemovePolicyRestoresAccess(t *testing.T) {
	e := NewEngine([]Policy{
		{ID: "allow-all", Effect: "permit", Principal: any(), Action: any(), Resource: any()},
		{ID: "block-delete", Effect: "forbid", Principal: any(), Action: eq("DELETE"), Resource: any()},
	})

	d := e.Evaluate(Request{
		Principal: map[string]string{"sub": "user"},
		Action:    "DELETE",
		Resource:  map[string]string{"index": "logs"},
	})
	if d.Allowed {
		t.Fatal("DELETE should be blocked")
	}

	e.RemovePolicy("block-delete")

	d2 := e.Evaluate(Request{
		Principal: map[string]string{"sub": "user"},
		Action:    "DELETE",
		Resource:  map[string]string{"index": "logs"},
	})
	if !d2.Allowed {
		t.Fatal("DELETE should be allowed after removing forbid policy")
	}
}
