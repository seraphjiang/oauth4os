package jwt

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/seraphjiang/oauth4os/internal/config"
)

type Claims struct {
	ClientID string
	Subject  string
	Issuer   string
	Scopes   []string
	Exp      time.Time
}

type Validator struct {
	providers []config.Provider
	jwksCache map[string]*jwksSet
	mu        sync.RWMutex
}

type jwksSet struct {
	Keys      []jwksKey
	FetchedAt time.Time
}

type jwksKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewValidator(providers []config.Provider) *Validator {
	return &Validator{
		providers: providers,
		jwksCache: make(map[string]*jwksSet),
	}
}

func (v *Validator) Validate(tokenStr string) (*Claims, error) {
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

	// Fetch JWKS and verify signature
	keys, err := v.getJWKS(provider)
	if err != nil {
		return nil, fmt.Errorf("JWKS fetch failed: %w", err)
	}

	kid, _ := token.Header["kid"].(string)
	key, err := findKey(keys, kid)
	if err != nil {
		return nil, err
	}

	// Re-parse with verification
	verified, err := jwtgo.Parse(tokenStr, func(t *jwtgo.Token) (interface{}, error) {
		return key, nil
	})
	if err != nil || !verified.Valid {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	verifiedClaims := verified.Claims.(jwtgo.MapClaims)

	// Extract scopes
	scopeStr, _ := verifiedClaims["scope"].(string)
	scopes := strings.Fields(scopeStr)

	clientID, _ := verifiedClaims["client_id"].(string)
	if clientID == "" {
		clientID, _ = verifiedClaims["azp"].(string)
	}
	sub, _ := verifiedClaims["sub"].(string)

	return &Claims{
		ClientID: clientID,
		Subject:  sub,
		Issuer:   issuer,
		Scopes:   scopes,
	}, nil
}

func (v *Validator) findProvider(issuer string) *config.Provider {
	for i := range v.providers {
		if v.providers[i].Issuer == issuer {
			return &v.providers[i]
		}
	}
	return nil
}

func (v *Validator) getJWKS(provider *config.Provider) ([]jwksKey, error) {
	v.mu.RLock()
	cached, ok := v.jwksCache[provider.Issuer]
	v.mu.RUnlock()

	if ok && time.Since(cached.FetchedAt) < 1*time.Hour {
		return cached.Keys, nil
	}

	jwksURI := provider.JWKSURI
	if jwksURI == "" || jwksURI == "auto" {
		jwksURI = strings.TrimSuffix(provider.Issuer, "/") + "/.well-known/openid-configuration"
		resp, err := http.Get(jwksURI)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var disc struct {
			JWKSURI string `json:"jwks_uri"`
		}
		json.NewDecoder(resp.Body).Decode(&disc)
		jwksURI = disc.JWKSURI
	}

	resp, err := http.Get(jwksURI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Keys []jwksKey `json:"keys"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	v.mu.Lock()
	v.jwksCache[provider.Issuer] = &jwksSet{Keys: result.Keys, FetchedAt: time.Now()}
	v.mu.Unlock()

	return result.Keys, nil
}

func findKey(keys []jwksKey, kid string) (*rsa.PublicKey, error) {
	for _, k := range keys {
		if k.Kid == kid && k.Kty == "RSA" {
			return parseRSAKey(k)
		}
	}
	if len(keys) > 0 && keys[0].Kty == "RSA" {
		return parseRSAKey(keys[0])
	}
	return nil, errors.New("no matching key found")
}

func parseRSAKey(k jwksKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}
