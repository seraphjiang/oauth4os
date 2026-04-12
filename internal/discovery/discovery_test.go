package discovery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_ReturnsValidMetadata(t *testing.T) {
	h := Handler(Config{Issuer: "https://proxy.example.com"}, []string{"read:logs-*", "admin"})
	req := httptest.NewRequest("GET", "/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var meta Metadata
	if err := json.Unmarshal(rec.Body.Bytes(), &meta); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if meta.Issuer != "https://proxy.example.com" {
		t.Fatalf("issuer = %s", meta.Issuer)
	}
	if meta.TokenEndpoint != "https://proxy.example.com/oauth/token" {
		t.Fatalf("token_endpoint = %s", meta.TokenEndpoint)
	}
	if meta.JWKSURI != "https://proxy.example.com/.well-known/jwks.json" {
		t.Fatalf("jwks_uri = %s", meta.JWKSURI)
	}
	if len(meta.ScopesSupported) != 2 {
		t.Fatalf("scopes = %v", meta.ScopesSupported)
	}
	if len(meta.CodeChallengeMethods) != 1 || meta.CodeChallengeMethods[0] != "S256" {
		t.Fatalf("code_challenge_methods = %v", meta.CodeChallengeMethods)
	}
}

func TestHandler_BaseURLOverride(t *testing.T) {
	h := Handler(Config{Issuer: "https://proxy.example.com", BaseURL: "https://cdn.example.com/auth"}, nil)
	req := httptest.NewRequest("GET", "/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	var meta Metadata
	json.Unmarshal(rec.Body.Bytes(), &meta)
	if meta.Issuer != "https://proxy.example.com" {
		t.Fatalf("issuer should be Issuer, got %s", meta.Issuer)
	}
	if meta.TokenEndpoint != "https://cdn.example.com/auth/oauth/token" {
		t.Fatalf("token_endpoint should use BaseURL, got %s", meta.TokenEndpoint)
	}
}

func TestHandler_CacheHeader(t *testing.T) {
	h := Handler(Config{Issuer: "https://proxy.example.com"}, nil)
	req := httptest.NewRequest("GET", "/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=3600" {
		t.Fatalf("Cache-Control = %s", cc)
	}
}
