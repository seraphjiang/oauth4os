package cedar

import "testing"

func TestTenantEngineUsesGlobalFallback(t *testing.T) {
	te := NewTenantEngine([]Policy{
		{ID: "global-permit", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
	})
	d := te.Evaluate("https://unknown.example.com", Request{
		Principal: map[string]string{"sub": "test"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs"},
	})
	if !d.Allowed || d.Policy != "global-permit" {
		t.Fatalf("expected global permit, got %+v", d)
	}
}

func TestTenantEngineUsesTenantPolicies(t *testing.T) {
	te := NewTenantEngine([]Policy{
		{ID: "global-permit", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
	})
	te.AddTenant("https://keycloak.example.com", []Policy{
		{ID: "tenant-forbid-delete", Effect: Forbid, Principal: Match{Any: true}, Action: Match{Equals: "DELETE"}, Resource: Match{Any: true}},
		{ID: "tenant-permit", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
	})

	// Tenant issuer → tenant policies (DELETE forbidden)
	d := te.Evaluate("https://keycloak.example.com", Request{
		Principal: map[string]string{"sub": "test"},
		Action:    "DELETE",
		Resource:  map[string]string{"index": "logs"},
	})
	if d.Allowed {
		t.Fatalf("expected tenant forbid on DELETE, got %+v", d)
	}

	// Tenant issuer → tenant policies (GET allowed)
	d = te.Evaluate("https://keycloak.example.com", Request{
		Principal: map[string]string{"sub": "test"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs"},
	})
	if !d.Allowed {
		t.Fatalf("expected tenant permit on GET, got %+v", d)
	}

	// Other issuer → global (DELETE allowed)
	d = te.Evaluate("https://other.example.com", Request{
		Principal: map[string]string{"sub": "test"},
		Action:    "DELETE",
		Resource:  map[string]string{"index": "logs"},
	})
	if !d.Allowed {
		t.Fatalf("expected global permit on DELETE for other issuer, got %+v", d)
	}
}
