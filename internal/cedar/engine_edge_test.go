package cedar

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentEvaluate(t *testing.T) {
	e := NewEngine([]Policy{
		{ID: "p1", Effect: "permit", Principal: "*", Action: "GET", Resource: "logs-*"},
		{ID: "p2", Effect: "forbid", Principal: "attacker", Action: "*", Resource: "*"},
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			req := Request{
				Principal: map[string]string{"sub": fmt.Sprintf("user-%d", n)},
				Action:    "GET",
				Resource:  map[string]string{"index": "logs-app"},
			}
			d := e.Evaluate(req)
			if !d.Allowed {
				t.Errorf("user-%d should be allowed", n)
			}
		}(i)
	}
	wg.Wait()
}

func TestConcurrentAddAndEvaluate(t *testing.T) {
	e := NewEngine([]Policy{
		{ID: "base", Effect: "permit", Principal: "*", Action: "*", Resource: "*"},
	})

	var wg sync.WaitGroup
	// Writers add policies
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			e.AddPolicy(Policy{
				ID: fmt.Sprintf("dyn-%d", n), Effect: "permit",
				Principal: fmt.Sprintf("svc-%d", n), Action: "GET", Resource: "logs-*",
			})
		}(i)
	}
	// Readers evaluate concurrently
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
		{ID: "p1", Effect: "permit", Principal: "*", Action: "*", Resource: "*"},
		{ID: "p2", Effect: "permit", Principal: "admin", Action: "*", Resource: "*"},
		{ID: "p3", Effect: "forbid", Principal: "admin", Action: "DELETE", Resource: "prod-*"},
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
		{ID: "allow-all", Effect: "permit", Principal: "*", Action: "*", Resource: "*"},
		{ID: "block-delete", Effect: "forbid", Principal: "*", Action: "DELETE", Resource: "*"},
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
