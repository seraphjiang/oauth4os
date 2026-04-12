package jwt

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/seraphjiang/oauth4os/internal/config"
)

// Claims holds validated JWT claims.
type Claims struct {
	ClientID string
	Subject  string
	Issuer   string
	Scopes   []string
	Exp      time.Time
}

// Validator validates JWTs against OIDC providers.
type Validator struct {
	providers []config.Provider
	jwksCache map[string]*jwksSet
	mu        sync.RWMutex
	client    *http.Client
}

type jwksSet struct {
	Keys      []jwksKey
	FetchedAt time.Time
}

type jwksKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// NewValidator creates a JWT validator with OIDC auto-discovery.
func NewValidator(providers []config.Provider) *Validator {
	return &Validator{
		providers: providers,
		jwksCache: make(map[string]*jwksSet),
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Validate verifies a JWT token and returns claims.
func (v *Validator) Validate(tokenStr string) (*Claims, error) {
	if tokenStr == "" {
		return nil, errors.New("empty token")
	}

	// Parse without verification to extract issuer for provider lookup
	parser := jwtgo.NewParser(jwtgo.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenStr, jwtgo.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("malformed token: %w", err)
	}

	mapClaims, ok := token.Claims.(jwtgo.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	issuer, _ := mapClaims["iss"].(string)
	provider := v.findProvider(issuer)
	if provider == nil {
		return nil, fmt.Errorf("unknown issuer: %s", issuer)
	}

	kid, _ := token.Header["kid"].(string)

	// Fetch JWKS and find signing key
	key, err := v.resolveKey(provider, kid)
	if err != nil {
		return nil, err
	}

	// Re-parse with full verification (signature + expiry)
	verified, err := jwtgo.Parse(tokenStr, func(t *jwtgo.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtgo.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return key, nil
	}, jwtgo.WithExpirationRequired())
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}
	if !verified.Valid {
		return nil, errors.New("invalid token")
	}

	verifiedClaims := verified.Claims.(jwtgo.MapClaims)

	// Validate audience if provider specifies expected values
	if len(provider.Audience) > 0 {
		tokenAud, _ := verifiedClaims.GetAudience()
		if !audienceMatch(tokenAud, provider.Audience) {
			return nil, fmt.Errorf("audience mismatch: token has %v, expected one of %v", tokenAud, provider.Audience)
		}
	}

	// Extract scopes (support both space-delimited string and array)
	scopes := extractScopes(verifiedClaims)

	clientID, _ := verifiedClaims["client_id"].(string)
	if clientID == "" {
		clientID, _ = verifiedClaims["azp"].(string)
	}
	sub, _ := verifiedClaims["sub"].(string)

	var exp time.Time
	if e, err := verifiedClaims.GetExpirationTime(); err == nil && e != nil {
		exp = e.Time
	}

	return &Claims{
		ClientID: clientID,
		Subject:  sub,
		Issuer:   issuer,
		Scopes:   scopes,
		Exp:      exp,
	}, nil
}

func extractScopes(claims jwtgo.MapClaims) []string {
	switch v := claims["scope"].(type) {
	case string:
		return strings.Fields(v)
	case []interface{}:
		var scopes []string
		for _, s := range v {
			if str, ok := s.(string); ok {
				scopes = append(scopes, str)
			}
		}
		return scopes
	}
	return nil
}

func (v *Validator) findProvider(issuer string) *config.Provider {
	for i := range v.providers {
		if v.providers[i].Issuer == issuer {
			return &v.providers[i]
		}
	}
	return nil
}

// resolveKey fetches the signing key, with retry on cache miss (handles key rotation).
func (v *Validator) resolveKey(provider *config.Provider, kid string) (*rsa.PublicKey, error) {
	keys, err := v.getJWKS(provider, false)
	if err != nil {
		return nil, fmt.Errorf("JWKS fetch failed: %w", err)
	}

	key, err := findKey(keys, kid)
	if err != nil && kid != "" {
		// Key not found — might be rotation. Force refresh and retry once.
		keys, err = v.getJWKS(provider, true)
		if err != nil {
			return nil, fmt.Errorf("JWKS refresh failed: %w", err)
		}
		return findKey(keys, kid)
	}
	return key, err
}

func (v *Validator) getJWKS(provider *config.Provider, forceRefresh bool) ([]jwksKey, error) {
	if !forceRefresh {
		v.mu.RLock()
		cached, ok := v.jwksCache[provider.Issuer]
		v.mu.RUnlock()
		if ok && time.Since(cached.FetchedAt) < 1*time.Hour {
			return cached.Keys, nil
		}
	}

	jwksURI, err := v.resolveJWKSURI(provider)
	if err != nil {
		return v.fallbackCache(provider.Issuer, err)
	}

	resp, err := v.client.Get(jwksURI)
	if err != nil {
		return v.fallbackCache(provider.Issuer, fmt.Errorf("JWKS request failed: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return v.fallbackCache(provider.Issuer, fmt.Errorf("JWKS endpoint returned %d", resp.StatusCode))
	}

	var result struct {
		Keys []jwksKey `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("JWKS decode failed: %w", err)
	}

	v.mu.Lock()
	v.jwksCache[provider.Issuer] = &jwksSet{Keys: result.Keys, FetchedAt: time.Now()}
	v.mu.Unlock()

	return result.Keys, nil
}

// fallbackCache returns stale cached JWKS (up to 24hr) when fetch fails.
// Logs warning but doesn't reject tokens — graceful degradation.
func (v *Validator) fallbackCache(issuer string, fetchErr error) ([]jwksKey, error) {
	v.mu.RLock()
	cached, ok := v.jwksCache[issuer]
	v.mu.RUnlock()
	if ok && time.Since(cached.FetchedAt) < 24*time.Hour {
		log.Printf("[WARN] JWKS fetch failed for %s, using stale cache (%s old): %v",
			issuer, time.Since(cached.FetchedAt).Round(time.Second), fetchErr)
		return cached.Keys, nil
	}
	return nil, fetchErr
}

// resolveJWKSURI uses OIDC discovery if jwks_uri is "auto" or empty.
func (v *Validator) resolveJWKSURI(provider *config.Provider) (string, error) {
	if provider.JWKSURI != "" && provider.JWKSURI != "auto" {
		return provider.JWKSURI, nil
	}

	discoveryURL := strings.TrimSuffix(provider.Issuer, "/") + "/.well-known/openid-configuration"
	resp, err := v.client.Get(discoveryURL)
	if err != nil {
		return "", fmt.Errorf("OIDC discovery failed for %s: %w", provider.Issuer, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC discovery returned %d for %s", resp.StatusCode, provider.Issuer)
	}

	var disc struct {
		JWKSURI string `json:"jwks_uri"`
		Issuer  string `json:"issuer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&disc); err != nil {
		return "", fmt.Errorf("OIDC discovery decode failed: %w", err)
	}

	// Validate issuer matches (prevents SSRF via discovery redirect)
	if disc.Issuer != provider.Issuer {
		return "", fmt.Errorf("issuer mismatch in discovery: expected %s, got %s", provider.Issuer, disc.Issuer)
	}

	if disc.JWKSURI == "" {
		return "", fmt.Errorf("no jwks_uri in discovery document for %s", provider.Issuer)
	}

	// Cache the resolved URI
	provider.JWKSURI = disc.JWKSURI
	return disc.JWKSURI, nil
}

func findKey(keys []jwksKey, kid string) (*rsa.PublicKey, error) {
	for _, k := range keys {
		if k.Kid == kid && k.Kty == "RSA" {
			return parseRSAKey(k)
		}
	}
	// Fallback: if only one RSA key and no kid match, use it
	if kid == "" {
		for _, k := range keys {
			if k.Kty == "RSA" {
				return parseRSAKey(k)
			}
		}
	}
	return nil, fmt.Errorf("no matching RSA key for kid=%s", kid)
}

// audienceMatch checks if any token audience is in the expected list.
func audienceMatch(tokenAud, expected []string) bool {
	for _, t := range tokenAud {
		for _, e := range expected {
			if t == e {
				return true
			}
		}
	}
	return false
}

func parseRSAKey(k jwksKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("invalid RSA N: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("invalid RSA E: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}
