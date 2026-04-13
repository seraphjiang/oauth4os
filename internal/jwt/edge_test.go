package jwt

import "testing"

// Edge: empty token fails validation
func TestEdge_EmptyTokenFails(t *testing.T) {
	v := NewValidator(nil)
	_, err := v.Validate("")
	if err == nil {
		t.Error("empty token should fail")
	}
}

// Edge: malformed token fails
func TestEdge_MalformedTokenFails(t *testing.T) {
	v := NewValidator(nil)
	_, err := v.Validate("not.a.valid.jwt.token")
	if err == nil {
		t.Error("malformed token should fail")
	}
}

// Edge: random string fails
func TestEdge_RandomStringFails(t *testing.T) {
	v := NewValidator(nil)
	_, err := v.Validate("random-garbage-string")
	if err == nil {
		t.Error("random string should fail")
	}
}
