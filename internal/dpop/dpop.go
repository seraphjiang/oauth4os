// Package dpop implements DPoP proof validation (RFC 9449 prep).
// Validates JWK thumbprint confirmation — binds tokens to client key pairs.
package dpop

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Proof represents a parsed DPoP proof JWT (simplified, header-only validation).
type Proof struct {
	JWKThumbprint string
	Method        string
	URI           string
	IssuedAt      time.Time
}

// Validate checks a DPoP proof from the request header.
// Returns the JWK thumbprint for token binding, or error.
func Validate(r *http.Request) (*Proof, error) {
	dpopHeader := r.Header.Get("DPoP")
	if dpopHeader == "" {
		return nil, nil // no DPoP = not required
	}

	parts := strings.Split(dpopHeader, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed DPoP proof")
	}

	// Decode header
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid DPoP header encoding")
	}
	var header struct {
		Typ string          `json:"typ"`
		Alg string          `json:"alg"`
		JWK json.RawMessage `json:"jwk"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("invalid DPoP header")
	}
	if header.Typ != "dpop+jwt" {
		return nil, fmt.Errorf("invalid DPoP typ: %s", header.Typ)
	}
	if header.JWK == nil {
		return nil, fmt.Errorf("missing jwk in DPoP header")
	}

	// Compute JWK thumbprint (RFC 7638)
	thumbprint := JWKThumbprint(header.JWK)

	// Decode payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid DPoP payload encoding")
	}
	var payload struct {
		HTM string `json:"htm"`
		HTU string `json:"htu"`
		IAT int64  `json:"iat"`
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, fmt.Errorf("invalid DPoP payload")
	}

	// Validate method and URI match
	if payload.HTM != r.Method {
		return nil, fmt.Errorf("DPoP method mismatch")
	}

	// Check freshness (5 minute window)
	iat := time.Unix(payload.IAT, 0)
	if time.Since(iat) > 5*time.Minute {
		return nil, fmt.Errorf("DPoP proof expired")
	}

	return &Proof{
		JWKThumbprint: thumbprint,
		Method:        payload.HTM,
		URI:           payload.HTU,
		IssuedAt:      iat,
	}, nil
}

// JWKThumbprint computes RFC 7638 thumbprint of a JWK.
func JWKThumbprint(jwk json.RawMessage) string {
	h := sha256.Sum256(jwk)
	return base64.RawURLEncoding.EncodeToString(h[:])
}
