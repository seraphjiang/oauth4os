package contract

import "testing"

func TestEdge_RunnerWithNoChecks(t *testing.T) {
	r := New("http://localhost:1")
	results := r.Run(nil)
	if len(results) != 0 {
		t.Error("nil checks should return empty results")
	}
}

func TestEdge_DefaultChecksHasHealth(t *testing.T) {
	checks := DefaultChecks()
	found := false
	for _, c := range checks {
		if c.Path == "/health" {
			found = true
		}
	}
	if !found {
		t.Error("DefaultChecks must include /health")
	}
}

func TestEdge_DefaultChecksHasOIDC(t *testing.T) {
	checks := DefaultChecks()
	found := false
	for _, c := range checks {
		if c.Path == "/.well-known/openid-configuration" {
			found = true
		}
	}
	if !found {
		t.Error("DefaultChecks must include OIDC discovery")
	}
}
