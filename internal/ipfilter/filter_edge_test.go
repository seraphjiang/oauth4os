package ipfilter

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentCheck(t *testing.T) {
	r, _ := New(Config{
		Global: &FilterConfig{Allow: []string{"10.0.0.0/8"}, Deny: []string{"10.0.0.99/32"}},
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ip := fmt.Sprintf("10.0.0.%d:1234", n%98+1) // 10.0.0.1-98
			err := r.Check("svc-1", ip)
			if n%98+1 == 99 {
				return // skip denied IP
			}
			if err != nil {
				t.Errorf("10.0.0.%d should be allowed: %v", n%98+1, err)
			}
		}(i)
	}
	wg.Wait()
}

func TestDenyOverridesAllow(t *testing.T) {
	r, _ := New(Config{
		Global: &FilterConfig{
			Allow: []string{"10.0.0.0/8"},
			Deny:  []string{"10.0.0.5/32"},
		},
	})

	if err := r.Check("", "10.0.0.5:1234"); err == nil {
		t.Fatal("10.0.0.5 should be denied even though 10.0.0.0/8 is allowed")
	}
	if err := r.Check("", "10.0.0.6:1234"); err != nil {
		t.Fatal("10.0.0.6 should be allowed")
	}
}

func TestPerClientOverride(t *testing.T) {
	r, _ := New(Config{
		Global: &FilterConfig{Allow: []string{"10.0.0.0/8"}},
		Clients: map[string]*FilterConfig{
			"restricted": {Allow: []string{"10.0.0.1/32"}},
		},
	})

	// restricted client can only use 10.0.0.1
	if err := r.Check("restricted", "10.0.0.1:1234"); err != nil {
		t.Fatal("restricted from 10.0.0.1 should be allowed")
	}
	if err := r.Check("restricted", "10.0.0.2:1234"); err == nil {
		t.Fatal("restricted from 10.0.0.2 should be denied")
	}
	// other clients use global
	if err := r.Check("other", "10.0.0.2:1234"); err != nil {
		t.Fatal("other client from 10.0.0.2 should be allowed by global")
	}
}

func TestIPv6(t *testing.T) {
	r, _ := New(Config{
		Global: &FilterConfig{Allow: []string{"::1/128"}},
	})

	if err := r.Check("", "[::1]:1234"); err != nil {
		t.Fatalf("IPv6 loopback should be allowed: %v", err)
	}
}

func TestInvalidCIDRReturnsError(t *testing.T) {
	_, err := New(Config{
		Global: &FilterConfig{Allow: []string{"not-a-cidr"}},
	})
	if err == nil {
		t.Fatal("invalid CIDR should return error")
	}
}
