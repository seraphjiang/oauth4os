package keyring

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew_GeneratesKey(t *testing.T) {
	r, err := New(2048, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	k := r.Current()
	if k == nil {
		t.Fatal("no current key")
	}
	if k.ID == "" {
		t.Fatal("empty kid")
	}
	if k.Private == nil {
		t.Fatal("nil private key")
	}
}

func TestRotate_PromotesPrevious(t *testing.T) {
	r, _ := New(2048, time.Hour)
	defer r.Stop()
	first := r.Current()
	r.rotate()
	second := r.Current()
	if first.ID == second.ID {
		t.Fatal("rotation should produce new key")
	}
	r.mu.RLock()
	prev := r.previous
	r.mu.RUnlock()
	if prev == nil || prev.ID != first.ID {
		t.Fatal("previous should be the old current")
	}
}

func TestJWKSHandler_ReturnsKeys(t *testing.T) {
	r, _ := New(2048, time.Hour)
	defer r.Stop()
	r.rotate() // now we have current + previous

	handler := r.JWKSHandler()
	req := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %s", ct)
	}

	var result jwks
	json.Unmarshal(rec.Body.Bytes(), &result)
	if len(result.Keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(result.Keys))
	}
	for _, k := range result.Keys {
		if k.Kty != "RSA" || k.Alg != "RS256" || k.Use != "sig" {
			t.Fatalf("bad key: %+v", k)
		}
		if k.N == "" || k.E == "" || k.Kid == "" {
			t.Fatal("missing key fields")
		}
	}
}

func TestJWKSHandler_CacheHeader(t *testing.T) {
	r, _ := New(2048, time.Hour)
	defer r.Stop()
	rec := httptest.NewRecorder()
	r.JWKSHandler()(rec, httptest.NewRequest("GET", "/", nil))
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=300" {
		t.Fatalf("Cache-Control = %s", cc)
	}
}

func TestAutoRotation(t *testing.T) {
	r, _ := New(2048, 50*time.Millisecond)
	defer r.Stop()
	first := r.Current().ID
	time.Sleep(120 * time.Millisecond)
	second := r.Current().ID
	if first == second {
		t.Fatal("key should have rotated")
	}
}
