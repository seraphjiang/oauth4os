package ratelimit

import "testing"

func TestEdge_ZeroDefaultRPM(t *testing.T) {
	l := New(nil, 0)
	// Should default to 600 RPM
	if !l.Allow("client", nil) {
		t.Error("should allow with default RPM")
	}
}

func TestEdge_NegativeDefaultRPM(t *testing.T) {
	l := New(nil, -1)
	if !l.Allow("client", nil) {
		t.Error("negative RPM should default to 600")
	}
}

func TestEdge_MultipleScopes(t *testing.T) {
	l := New(map[string]int{"admin": 10, "read": 1000}, 600)
	// Should use most restrictive (admin=10)
	scopes := []string{"read", "admin"}
	for i := 0; i < 10; i++ {
		l.Allow("c", scopes)
	}
	if l.Allow("c", scopes) {
		t.Error("should be limited by most restrictive scope")
	}
}
