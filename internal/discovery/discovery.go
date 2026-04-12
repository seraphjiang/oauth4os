// Package discovery implements OpenID Connect Discovery 1.0.
// Serves /.well-known/openid-configuration so clients can auto-discover
// the proxy's token, introspection, and JWKS endpoints.
package discovery

import (
	"encoding/json"
	"net/http"
)

// Config holds the values needed to build the discovery document.
type Config struct {
	Issuer  string // e.g. https://oauth4os.example.com
	BaseURL string // same as Issuer unless behind a path prefix
}

// Metadata is the OpenID Connect Discovery 1.0 response.
type Metadata struct {
	Issuer                string   `json:"issuer"`
	TokenEndpoint         string   `json:"token_endpoint"`
	IntrospectionEndpoint string   `json:"introspection_endpoint"`
	RevocationEndpoint    string   `json:"revocation_endpoint"`
	JWKSURI               string   `json:"jwks_uri"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	ResponseTypesSupported []string `json:"response_types_supported"`
	GrantTypesSupported   []string `json:"grant_types_supported"`
	TokenEndpointAuth     []string `json:"token_endpoint_auth_methods_supported"`
	ScopesSupported       []string `json:"scopes_supported,omitempty"`
	CodeChallengeMethods  []string `json:"code_challenge_methods_supported"`
	IntrospectionAuth     []string `json:"introspection_endpoint_auth_methods_supported"`
}

// Handler returns an http.HandlerFunc for /.well-known/openid-configuration.
func Handler(cfg Config, scopes []string) http.HandlerFunc {
	base := cfg.BaseURL
	if base == "" {
		base = cfg.Issuer
	}
	meta := Metadata{
		Issuer:                cfg.Issuer,
		TokenEndpoint:         base + "/oauth/token",
		IntrospectionEndpoint: base + "/oauth/introspect",
		RevocationEndpoint:    base + "/oauth/revoke",
		JWKSURI:               base + "/.well-known/jwks.json",
		AuthorizationEndpoint: base + "/oauth/authorize",
		ResponseTypesSupported: []string{"code"},
		GrantTypesSupported:   []string{"client_credentials", "authorization_code", "urn:ietf:params:oauth:grant-type:token-exchange"},
		TokenEndpointAuth:     []string{"client_secret_post", "none"},
		ScopesSupported:       scopes,
		CodeChallengeMethods:  []string{"S256"},
		IntrospectionAuth:     []string{"client_secret_post"},
	}
	body, _ := json.MarshalIndent(meta, "", "  ")

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(body)
	}
}
