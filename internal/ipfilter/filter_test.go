package ipfilter

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func mustNew(t *testing.T, cfg Config) *Rules {
	t.Helper()
	r, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestAllowlist_Permits(t *testing.T) {
	r := mustNew(t, Config{Clients: map[string]*FilterConfig{
		"agent": {Allow: []string{"10.0.0.0/8"}},
	}})
	if err := r.Check("agent", "10.1.2.3:1234"); err != nil {
		t.Fatalf("should allow: %v", err)
	}
}

func TestAllowlist_Denies(t *testing.T) {
	r := mustNew(t, Config{Clients: map[string]*FilterConfig{
		"agent": {Allow: []string{"10.0.0.0/8"}},
	}})
	if err := r.Check("agent", "192.168.1.1:1234"); err == nil {
		t.Fatal("should deny IP outside allowlist")
	}
}

func TestDenylist_Blocks(t *testing.T) {
	r := mustNew(t, Config{Clients: map[string]*FilterConfig{
		"agent": {Deny: []string{"192.168.0.0/16"}},
	}})
	if err := r.Check("agent", "192.168.1.1:1234"); err == nil {
		t.Fatal("should deny")
	}
}

func TestDenylist_AllowsOther(t *testing.T) {
	r := mustNew(t, Config{Clients: map[string]*FilterConfig{
		"agent": {Deny: []string{"192.168.0.0/16"}},
	}})
	if err := r.Check("agent", "10.0.0.1:1234"); err != nil {
		t.Fatalf("should allow: %v", err)
	}
}

func TestDeny_OverridesAllow(t *testing.T) {
	r := mustNew(t, Config{Clients: map[string]*FilterConfig{
		"agent": {Allow: []string{"10.0.0.0/8"}, Deny: []string{"10.0.0.5/32"}},
	}})
	if err := r.Check("agent", "10.0.0.5:1234"); err == nil {
		t.Fatal("deny should override allow")
	}
	if err := r.Check("agent", "10.0.0.6:1234"); err != nil {
		t.Fatalf("other IPs should be allowed: %v", err)
	}
}

func TestGlobal_Fallback(t *testing.T) {
	r := mustNew(t, Config{Global: &FilterConfig{Allow: []string{"10.0.0.0/8"}}})
	if err := r.Check("unknown-client", "10.1.1.1:80"); err != nil {
		t.Fatalf("global should apply: %v", err)
	}
	if err := r.Check("unknown-client", "172.16.0.1:80"); err == nil {
		t.Fatal("global should deny")
	}
}

func TestNoRules_AllowsAll(t *testing.T) {
	r := mustNew(t, Config{})
	if err := r.Check("any", "1.2.3.4:80"); err != nil {
		t.Fatalf("no rules should allow all: %v", err)
	}
}

func TestSingleIP_NoCIDR(t *testing.T) {
	r := mustNew(t, Config{Clients: map[string]*FilterConfig{
		"agent": {Allow: []string{"10.0.0.1"}},
	}})
	if err := r.Check("agent", "10.0.0.1:80"); err != nil {
		t.Fatalf("single IP should work: %v", err)
	}
}

func TestMiddleware_Blocks(t *testing.T) {
	r := mustNew(t, Config{Global: &FilterConfig{Deny: []string{"0.0.0.0/0"}}})
	handler := r.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), func(r *http.Request) string { return "client" })

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:80"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestInvalidCIDR(t *testing.T) {
	_, err := New(Config{Global: &FilterConfig{Allow: []string{"not-a-cidr"}}})
	if err == nil {
		t.Fatal("should reject invalid CIDR")
	}
}
