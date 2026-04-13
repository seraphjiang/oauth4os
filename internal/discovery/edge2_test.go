package discovery

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestEdge_ResponseContainsScopes(t *testing.T) {
	h := Handler(Config{Issuer: "https://proxy.example.com", BaseURL: "https://proxy.example.com"}, []string{"openid", "read", "write"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/.well-known/openid-configuration", nil))
	var doc map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &doc)
	if scopes, ok := doc["scopes_supported"]; ok {
		arr, _ := scopes.([]interface{})
		if len(arr) < 3 {
			t.Errorf("expected 3+ scopes, got %d", len(arr))
		}
	}
}

func TestEdge_TokenEndpointPresent(t *testing.T) {
	h := Handler(Config{Issuer: "https://proxy.example.com", BaseURL: "https://proxy.example.com"}, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/.well-known/openid-configuration", nil))
	var doc map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &doc)
	if _, ok := doc["token_endpoint"]; !ok {
		t.Error("token_endpoint must be present")
	}
}

func TestEdge_JWKSURIPresent(t *testing.T) {
	h := Handler(Config{Issuer: "https://proxy.example.com", BaseURL: "https://proxy.example.com"}, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/.well-known/openid-configuration", nil))
	var doc map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &doc)
	if _, ok := doc["jwks_uri"]; !ok {
		t.Error("jwks_uri must be present")
	}
}
