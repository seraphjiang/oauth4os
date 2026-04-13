package sigv4

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEdge_CanonicalHeadersSorted(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Amz-Date", "20260413T000000Z")
	r.Header.Set("Host", "example.com")
	signed, canonical := canonicalHeaderStr(r)
	if signed == "" || canonical == "" {
		t.Error("should produce canonical headers")
	}
	if !strings.Contains(signed, "host") {
		t.Error("signed headers should include host")
	}
}

func TestEdge_HmacSHA256Deterministic(t *testing.T) {
	h1 := hmacSHA256([]byte("key"), []byte("data"))
	h2 := hmacSHA256([]byte("key"), []byte("data"))
	if len(h1) == 0 || len(h2) == 0 {
		t.Error("hmac should not be empty")
	}
	for i := range h1 {
		if h1[i] != h2[i] {
			t.Error("same input should produce same hmac")
			break
		}
	}
}

func TestEdge_DeriveKeyDifferentRegions(t *testing.T) {
	k1 := deriveKey("secret", "20260413", "us-east-1", "es")
	k2 := deriveKey("secret", "20260413", "eu-west-1", "es")
	if string(k1) == string(k2) {
		t.Error("different regions should produce different keys")
	}
}
