package discovery

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// Mutation: remove issuer → discovery must include issuer
func TestMutation_IssuerPresent(t *testing.T) {
	h := Handler(Config{Issuer: "https://proxy.example.com"}, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/.well-known/openid-configuration", nil))
	var m Metadata
	json.Unmarshal(w.Body.Bytes(), &m)
	if m.Issuer != "https://proxy.example.com" {
		t.Errorf("issuer should be https://proxy.example.com, got %s", m.Issuer)
	}
}

// Mutation: remove JWKS URI → must include jwks_uri
func TestMutation_JWKSURI(t *testing.T) {
	h := Handler(Config{Issuer: "https://proxy.example.com"}, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	var m Metadata
	json.Unmarshal(w.Body.Bytes(), &m)
	if m.JWKSURI == "" {
		t.Error("jwks_uri must be present")
	}
}

// Mutation: remove Cache-Control → must cache discovery document
func TestMutation_CacheControl(t *testing.T) {
	h := Handler(Config{Issuer: "https://proxy.example.com"}, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Header().Get("Cache-Control") == "" {
		t.Error("discovery must set Cache-Control header")
	}
}

// Mutation: remove Content-Type → must return application/json
func TestMutation_ContentType(t *testing.T) {
	h := Handler(Config{Issuer: "https://proxy.example.com"}, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("discovery must return application/json")
	}
}
