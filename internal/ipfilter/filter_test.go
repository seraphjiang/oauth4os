package ipfilter

import "testing"

func TestAllowOnly(t *testing.T) {
	f := New(map[string]Rule{
		"agent-1": {Allow: []string{"10.0.0.0/8"}},
	})
	if !f.Check("agent-1", "10.1.2.3:1234") {
		t.Error("10.1.2.3 should be allowed")
	}
	if f.Check("agent-1", "192.168.1.1:1234") {
		t.Error("192.168.1.1 should be denied")
	}
}

func TestDenyOverridesAllow(t *testing.T) {
	f := New(map[string]Rule{
		"agent-1": {Allow: []string{"10.0.0.0/8"}, Deny: []string{"10.0.0.5"}},
	})
	if !f.Check("agent-1", "10.0.0.1:80") {
		t.Error("10.0.0.1 should be allowed")
	}
	if f.Check("agent-1", "10.0.0.5:80") {
		t.Error("10.0.0.5 should be denied")
	}
}

func TestGlobalRule(t *testing.T) {
	f := New(map[string]Rule{
		"*": {Deny: []string{"192.168.0.0/16"}},
	})
	if f.Check("any-client", "192.168.1.1:80") {
		t.Error("192.168.1.1 should be denied globally")
	}
	if !f.Check("any-client", "10.0.0.1:80") {
		t.Error("10.0.0.1 should be allowed")
	}
}

func TestNoRulesAllowAll(t *testing.T) {
	f := New(map[string]Rule{})
	if !f.Check("unknown", "1.2.3.4:80") {
		t.Error("no rules should allow all")
	}
}

func TestBareIP(t *testing.T) {
	f := New(map[string]Rule{
		"x": {Allow: []string{"1.2.3.4"}},
	})
	if !f.Check("x", "1.2.3.4:80") {
		t.Error("exact IP should match")
	}
	if f.Check("x", "1.2.3.5:80") {
		t.Error("different IP should not match")
	}
}

func TestClientSpecificOverridesGlobal(t *testing.T) {
	f := New(map[string]Rule{
		"*":       {Deny: []string{"10.0.0.0/8"}},
		"trusted": {Allow: []string{"10.0.0.0/8"}},
	})
	// Global denies 10.x for other clients
	if f.Check("other", "10.0.0.1:80") {
		t.Error("global deny should block other clients")
	}
	// Client-specific allow overrides for trusted
	if !f.Check("trusted", "10.0.0.1:80") {
		t.Error("client-specific allow should override")
	}
}
