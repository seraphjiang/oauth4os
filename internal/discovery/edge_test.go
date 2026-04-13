package discovery

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// Edge: OIDC discovery returns required fields
func TestEdge_OIDCRequiredFields(t *testing.T) {
	h := Handler(Config{Issuer: "https://proxy.example.com", BaseURL: "https://proxy.example.com"}, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/.well-known/openid-configuration", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var doc map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &doc)
	for _, field := range []string{"issuer", "jwks_uri", "token_endpoint"} {
		if _, ok := doc[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}
}

// Edge: issuer matches configured value
func TestEdge_IssuerMatches(t *testing.T) {
	h := Handler(Config{Issuer: "https://my-proxy.example.com", BaseURL: "https://my-proxy.example.com"}, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/.well-known/openid-configuration", nil))
	var doc map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &doc)
	if doc["issuer"] != "https://my-proxy.example.com" {
		t.Errorf("issuer should match, got %v", doc["issuer"])
	}
}
