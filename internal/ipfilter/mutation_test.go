package ipfilter

import (
	"testing"
)

// Mutation: remove allowlist check → allowed IP must pass
func TestMutation_AllowlistPass(t *testing.T) {
	r, err := New(Config{Filters: []FilterConfig{{ClientID: "app", Allow: []string{"10.0.0.0/8"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Check("app", "10.1.2.3:1234"); err != nil {
		t.Errorf("allowed IP should pass: %v", err)
	}
}

// Mutation: remove denylist check → denied IP must be rejected
func TestMutation_DenylistBlock(t *testing.T) {
	r, err := New(Config{Filters: []FilterConfig{{ClientID: "app", Deny: []string{"192.168.0.0/16"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Check("app", "192.168.1.1:1234"); err == nil {
		t.Error("denied IP should be rejected")
	}
}

// Mutation: remove IP extraction → must handle addr:port format
func TestMutation_IPExtraction(t *testing.T) {
	ip := extractIP("203.0.113.5:8080")
	if ip == nil {
		t.Error("must extract IP from addr:port")
	}
	if ip.String() != "203.0.113.5" {
		t.Errorf("expected 203.0.113.5, got %s", ip.String())
	}
}

// Mutation: unconfigured client → must pass (no rules = allow all)
func TestMutation_NoRulesAllow(t *testing.T) {
	r, err := New(Config{Filters: []FilterConfig{{ClientID: "other", Allow: []string{"10.0.0.0/8"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Check("unknown-client", "1.2.3.4:80"); err != nil {
		t.Errorf("unconfigured client should be allowed: %v", err)
	}
}
