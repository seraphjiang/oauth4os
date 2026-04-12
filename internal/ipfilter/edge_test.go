package ipfilter

import (
	"net"
	"testing"
	"testing/quick"
)

// Edge cases and chaos tests for IP filter.

func TestInvalidIP_Rejected(t *testing.T) {
	f := New(map[string]Rule{})
	if f.Check("c", "not-an-ip") {
		t.Error("invalid IP should be rejected")
	}
}

func TestEmptyRemoteAddr(t *testing.T) {
	f := New(map[string]Rule{"c": {Allow: []string{"10.0.0.0/8"}}})
	if f.Check("c", "") {
		t.Error("empty remote addr should be rejected")
	}
}

func TestIPv6(t *testing.T) {
	f := New(map[string]Rule{"c": {Allow: []string{"::1/128"}}})
	if !f.Check("c", "[::1]:8080") {
		t.Error("IPv6 loopback should be allowed")
	}
	if f.Check("c", "[::2]:8080") {
		t.Error("non-loopback IPv6 should be denied")
	}
}

func TestDenyTakesPrecedenceOverAllow(t *testing.T) {
	f := New(map[string]Rule{"c": {
		Allow: []string{"10.0.0.0/8"},
		Deny:  []string{"10.0.0.1/32"},
	}})
	if f.Check("c", "10.0.0.1:1234") {
		t.Error("denied IP should be rejected even if in allow range")
	}
	if !f.Check("c", "10.0.0.2:1234") {
		t.Error("non-denied IP in allow range should pass")
	}
}

func TestInvalidCIDR_Ignored(t *testing.T) {
	// Invalid CIDR should be silently ignored, not crash
	f := New(map[string]Rule{"c": {Allow: []string{"not-a-cidr", "10.0.0.0/8"}}})
	if !f.Check("c", "10.0.0.1:80") {
		t.Error("valid IP should pass despite invalid CIDR in list")
	}
}

func TestPortStripping(t *testing.T) {
	f := New(map[string]Rule{"c": {Allow: []string{"192.168.1.1/32"}}})
	if !f.Check("c", "192.168.1.1:443") {
		t.Error("should strip port before matching")
	}
	if !f.Check("c", "192.168.1.1:0") {
		t.Error("should work with port 0")
	}
}

func TestNoPortInAddr(t *testing.T) {
	f := New(map[string]Rule{"c": {Allow: []string{"1.2.3.4/32"}}})
	if !f.Check("c", "1.2.3.4") {
		t.Error("bare IP without port should work")
	}
}

// Property: deny list always blocks, regardless of allow list.
func TestProperty_DenyAlwaysBlocks(t *testing.T) {
	fn := func(lastOctet uint8) bool {
		ip := net.IPv4(10, 0, 0, lastOctet).String()
		f := New(map[string]Rule{"c": {
			Allow: []string{"0.0.0.0/0"}, // allow everything
			Deny:  []string{"10.0.0.0/8"}, // deny 10.x
		}})
		return !f.Check("c", ip+":80")
	}
	if err := quick.Check(fn, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}
