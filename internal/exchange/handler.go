// Package exchange implements RFC 8693 OAuth 2.0 Token Exchange.
// Allows clients to exchange external IdP tokens (subject_token) for
// oauth4os-scoped access tokens.
package exchange

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	GrantType          = "urn:ietf:params:oauth:grant-type:token-exchange"
	AccessTokenType    = "urn:ietf:params:oauth:token-type:access_token"
	IDTokenType        = "urn:ietf:params:oauth:token-type:id_token"
	JWTTokenType       = "urn:ietf:params:oauth:token-type:jwt"
)

// SubjectValidator validates the incoming subject_token and returns claims.
type SubjectValidator interface {
	ValidateSubject(token string) (*SubjectClaims, error)
}

// SubjectClaims are the validated claims from the external token.
type SubjectClaims struct {
	Subject  string   // sub claim
	Issuer   string   // iss claim
	Audience string   // aud claim
	Scopes   []string // scope claim (if present)
	Email    string   // email claim (if present)
}

// TokenIssuer issues oauth4os tokens.
type TokenIssuer interface {
	IssueExchangeToken(subject, issuer string, scopes []string) (tokenID string, expiresIn int)
}

// Handler handles POST /oauth/token with grant_type=urn:ietf:params:oauth:grant-type:token-exchange.
type Handler struct {
	validator SubjectValidator
	issuer    TokenIssuer
	audience  string // expected audience for this proxy
}

// NewHandler creates a token exchange handler.
func NewHandler(validator SubjectValidator, issuer TokenIssuer, audience string) *Handler {
	return &Handler{validator: validator, issuer: issuer, audience: audience}
}

// ServeHTTP implements RFC 8693 token exchange.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	subjectToken := r.FormValue("subject_token")
	subjectTokenType := r.FormValue("subject_token_type")
	actorToken := r.FormValue("actor_token")
	actorTokenType := r.FormValue("actor_token_type")
	requestedScope := r.FormValue("scope")

	if subjectToken == "" {
		writeErr(w, 400, "invalid_request", "subject_token is required")
		return
	}
	if subjectTokenType == "" {
		subjectTokenType = AccessTokenType
	}
	if subjectTokenType != AccessTokenType && subjectTokenType != IDTokenType && subjectTokenType != JWTTokenType {
		writeErr(w, 400, "invalid_request", "unsupported subject_token_type")
		return
	}

	// Validate the subject token
	claims, err := h.validator.ValidateSubject(subjectToken)
	if err != nil {
		writeErr(w, 401, "invalid_grant", "subject_token validation failed: "+err.Error())
		return
	}

	// Validate actor token if provided (delegation flow per RFC 8693 §2.1)
	var actorClaims *SubjectClaims
	if actorToken != "" {
		if actorTokenType == "" {
			actorTokenType = AccessTokenType
		}
		ac, err := h.validator.ValidateSubject(actorToken)
		if err != nil {
			writeErr(w, 401, "invalid_grant", "actor_token validation failed: "+err.Error())
			return
		}
		actorClaims = ac
	}

	// Determine scopes
	var scopes []string
	if requestedScope != "" {
		scopes = strings.Fields(requestedScope)
	} else if len(claims.Scopes) > 0 {
		scopes = claims.Scopes
	}

	// Issue oauth4os token
	tokenID, expiresIn := h.issuer.IssueExchangeToken(claims.Subject, claims.Issuer, scopes)

	resp := map[string]interface{}{
		"access_token":       tokenID,
		"issued_token_type":  AccessTokenType,
		"token_type":         "Bearer",
		"expires_in":         expiresIn,
		"scope":              strings.Join(scopes, " "),
	})
}

func writeErr(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": desc,
	})
}

// ManagerAdapter adapts token.Manager to TokenIssuer for exchange.
type ManagerAdapter struct {
	CreateToken func(clientID string, scopes []string) (tokenID string, refreshToken string)
}

func (a *ManagerAdapter) IssueExchangeToken(subject, issuer string, scopes []string) (string, int) {
	// Use subject@issuer as the client identity for exchanged tokens
	clientID := subject + "@" + issuer
	tokenID, _ := a.CreateToken(clientID, scopes)
	return tokenID, 3600
}

// JWTSubjectValidator validates subject tokens using the proxy's JWT validator.
type JWTSubjectValidator struct {
	Validate func(token string) (sub, iss string, scopes []string, err error)
}

func (v *JWTSubjectValidator) ValidateSubject(token string) (*SubjectClaims, error) {
	sub, iss, scopes, err := v.Validate(token)
	if err != nil {
		return nil, err
	}
	return &SubjectClaims{Subject: sub, Issuer: iss, Scopes: scopes}, nil
}

// StaticSubjectValidator always returns fixed claims. For testing.
type StaticSubjectValidator struct {
	Claims *SubjectClaims
	Err    error
}

func (v *StaticSubjectValidator) ValidateSubject(token string) (*SubjectClaims, error) {
	return v.Claims, v.Err
}

// StaticTokenIssuer returns a fixed token. For testing.
type StaticTokenIssuer struct {
	TokenID   string
	ExpiresIn int
	mu          sync.Mutex
	Called      int
	LastSubject string
	LastScopes  []string
}

func (i *StaticTokenIssuer) IssueExchangeToken(subject, issuer string, scopes []string) (string, int) {
	i.mu.Lock()
	i.Called++
	i.LastSubject = subject
	i.LastScopes = scopes
	i.mu.Unlock()
	return i.TokenID, i.ExpiresIn
}

// ExchangeTime is used for testing time-dependent behavior.
var ExchangeTime = func() time.Time { return time.Now() }
