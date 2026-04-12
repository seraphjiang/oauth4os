package tokenbind

import (
	"net/http/httptest"
	"sync"
	"testing"
)

func TestDifferentIPRejected(t *testing.T) {
	b := New()
	r1 := httptest.NewRequest("GET", "/", nil)
	r1.RemoteAddr = "10.0.0.1:1234"
	r1.Header.Set("User-Agent", "cli/1.0")
	b.Bind("tok-1", Fingerprint(r1))

	r2 := httptest.NewRequest("GET", "/", nil)
	r2.RemoteAddr = "10.0.0.2:5678"
	r2.Header.Set("User-Agent", "cli/1.0")
	if b.Verify("tok-1", Fingerprint(r2)) {
		t.Fatal("different IP should be rejected")
	}
}

func TestDifferentUARejected(t *testing.T) {
	b := New()
	r1 := httptest.NewRequest("GET", "/", nil)
	r1.RemoteAddr = "10.0.0.1:1234"
	r1.Header.Set("User-Agent", "cli/1.0")
	b.Bind("tok-1", Fingerprint(r1))

	r2 := httptest.NewRequest("GET", "/", nil)
	r2.RemoteAddr = "10.0.0.1:9999"
	r2.Header.Set("User-Agent", "browser/2.0")
	if b.Verify("tok-1", Fingerprint(r2)) {
		t.Fatal("different User-Agent should be rejected")
	}
}

func TestConcurrentBindAndVerify(t *testing.T) {
	b := New()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = "10.0.0.1:1234"
			r.Header.Set("User-Agent", "cli/1.0")
			fp := Fingerprint(r)
			if n%2 == 0 {
				b.Bind("tok-c", fp)
			} else {
				b.Verify("tok-c", fp)
			}
		}(i)
	}
	wg.Wait()
}

func TestRemoveAndRebind(t *testing.T) {
	b := New()
	r1 := httptest.NewRequest("GET", "/", nil)
	r1.RemoteAddr = "10.0.0.1:1234"
	r1.Header.Set("User-Agent", "cli/1.0")
	b.Bind("tok-1", Fingerprint(r1))
	b.Remove("tok-1")

	r2 := httptest.NewRequest("GET", "/", nil)
	r2.RemoteAddr = "10.0.0.2:5678"
	r2.Header.Set("User-Agent", "browser/2.0")
	b.Bind("tok-1", Fingerprint(r2))
	if !b.Verify("tok-1", Fingerprint(r2)) {
		t.Fatal("rebind after remove should work")
	}
}

func TestPortIgnored(t *testing.T) {
	b := New()
	r1 := httptest.NewRequest("GET", "/", nil)
	r1.RemoteAddr = "10.0.0.1:1234"
	r1.Header.Set("User-Agent", "cli/1.0")
	b.Bind("tok-1", Fingerprint(r1))

	r2 := httptest.NewRequest("GET", "/", nil)
	r2.RemoteAddr = "10.0.0.1:9999"
	r2.Header.Set("User-Agent", "cli/1.0")
	if !b.Verify("tok-1", Fingerprint(r2)) {
		t.Fatal("same IP different port should pass")
	}
}
