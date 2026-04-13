package ipfilter

import "testing"

// Edge: allowed IP passes check
func TestEdge_AllowedIPPasses(t *testing.T) {
	r, err := New(Config{
		Clients: map[string]*FilterConfig{
			"app": {Allow: []string{"10.0.0.0/8"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Check("app", "10.1.2.3:1234"); err != nil {
		t.Errorf("allowed IP should pass: %v", err)
	}
}

// Edge: denied IP fails check
func TestEdge_DeniedIPFails(t *testing.T) {
	r, err := New(Config{
		Clients: map[string]*FilterConfig{
			"app": {Allow: []string{"10.0.0.0/8"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Check("app", "192.168.1.1:1234"); err == nil {
		t.Error("denied IP should fail")
	}
}

// Edge: unknown client passes (no filter configured)
func TestEdge_UnknownClientPasses(t *testing.T) {
	r, err := New(Config{
		Clients: map[string]*FilterConfig{
			"app": {Allow: []string{"10.0.0.0/8"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Check("other-app", "192.168.1.1:1234"); err != nil {
		t.Errorf("unknown client should pass: %v", err)
	}
}
