package ipfilter

import (
	"sync"
	"testing"
)

func TestEdge_ConcurrentCheck(t *testing.T) {
	r, err := New(Config{
		Clients: map[string]*FilterConfig{
			"app": {Allow: []string{"10.0.0.0/8"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Check("app", "10.1.2.3:8080")
		}()
	}
	wg.Wait()
}

func TestEdge_IPv6Address(t *testing.T) {
	r, err := New(Config{
		Clients: map[string]*FilterConfig{
			"app": {Allow: []string{"::1/128"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Check("app", "[::1]:8080"); err != nil {
		t.Errorf("IPv6 loopback should be allowed: %v", err)
	}
}

func TestEdge_EmptyConfig(t *testing.T) {
	r, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	// No filters = allow all
	if err := r.Check("any", "1.2.3.4:80"); err != nil {
		t.Errorf("empty config should allow all: %v", err)
	}
}
