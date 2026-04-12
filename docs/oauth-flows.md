# OAuth Flows — oauth4os

All supported OAuth 2.0 flows with sequence diagrams.

## 1. Client Credentials (RFC 6749 §4.4)

Machine-to-machine authentication. The primary flow for AI agents, CI/CD pipelines, and backend services.

```mermaid
sequenceDiagram
    participant C as Client
    participant P as oauth4os Proxy
    participant O as OpenSearch

    C->>P: POST /oauth/token
    Note right of C: grant_type=client_credentials<br/>client_id=my-agent<br/>client_secret=***<br/>scope=read:logs-*
    P->>P: Authenticate client (constant-time compare)
    P->>P: Validate scopes against client allowance
    P->>P: Issue access_token + refresh_token
    P->>C: 200 {access_token, refresh_token, expires_in: 3600}
    C->>P: GET /logs-2025/_search<br/>Authorization: Bearer <token>
    P->>P: Validate JWT signature (JWKS)
    P->>P: Check expiry
    P->>P: Map scopes → OpenSearch roles
    P->>P: Cedar policy evaluation
    P->>O: Forward with X-Proxy-User, X-Proxy-Roles
    O->>P: Search results
    P->>C: Search results
```

**Config:**
```yaml
# Clients registered via POST /oauth/register or config
scope_mapping:
  "read:logs-*":
    backend_user: agent-logs-reader
    backend_roles: [logs_read_access]
```

**curl:**
```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8443/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=my-agent" \
  -d "client_secret=secret" \
  -d "scope=read:logs-*" | jq -r .access_token)

# Query
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/logs-*/_search \
  -d '{"query":{"match":{"level":"error"}}}'
```

## 2. PKCE Authorization Code (RFC 7636)

Browser-based authentication. Prevents authorization code interception without requiring a client secret.

```mermaid
sequenceDiagram
    participant B as Browser
    participant P as oauth4os Proxy

    B->>B: Generate code_verifier (random 43-128 chars)
    B->>B: code_challenge = BASE64URL(SHA256(code_verifier))
    B->>P: GET /oauth/authorize
    Note right of B: client_id=my-app<br/>code_challenge=...<br/>code_challenge_method=S256<br/>redirect_uri=http://localhost/cb<br/>scope=read:logs-*
    P->>P: Validate redirect_uri against client allowlist
    P->>P: Store auth code + challenge (10min TTL)
    P->>B: 302 → redirect_uri?code=AUTH_CODE
    B->>P: POST /oauth/authorize/token
    Note right of B: code=AUTH_CODE<br/>code_verifier=...<br/>redirect_uri=http://localhost/cb
    P->>P: SHA256(code_verifier) == code_challenge?
    P->>P: Constant-time compare
    P->>P: Delete code (one-time use)
    P->>P: Issue tokens
    P->>B: 200 {access_token, refresh_token}
```

**Security controls:**
- S256 only (plain not supported)
- Constant-time PKCE verification (`crypto/subtle`)
- One-time authorization codes
- 10-minute code expiry
- redirect_uri must match client's registered allowlist (prevents open redirect)

## 3. Token Refresh (RFC 6749 §6)

Rotate tokens without re-authentication. Includes reuse detection.

```mermaid
sequenceDiagram
    participant C as Client
    participant P as oauth4os Proxy

    C->>P: POST /oauth/token
    Note right of C: grant_type=refresh_token<br/>refresh_token=rtk_...<br/>client_id=my-agent<br/>client_secret=***
    P->>P: Authenticate client
    P->>P: Validate refresh token belongs to client
    P->>P: Revoke old access + refresh token
    P->>P: Mark old refresh token as "used"
    P->>P: Issue new access + refresh token
    P->>C: 200 {access_token, refresh_token}
```

**Reuse detection:**
```mermaid
sequenceDiagram
    participant A as Attacker (stolen token)
    participant P as oauth4os Proxy

    Note over A,P: Attacker replays a previously-used refresh token
    A->>P: POST /oauth/token (refresh_token=rtk_old)
    P->>P: Detect: rtk_old already used!
    P->>P: Revoke ALL tokens for this client
    P->>A: 400 {error: "invalid_grant", "refresh token reuse detected"}
```

If a refresh token is used twice, it indicates theft. The entire token family for that client is revoked immediately.

## 4. Token Exchange (RFC 8693)

Exchange an external OIDC token for an oauth4os token. Enables federation with Keycloak, Auth0, Okta, etc.

```mermaid
sequenceDiagram
    participant C as Client
    participant P as oauth4os Proxy
    participant IdP as OIDC Provider

    C->>IdP: Authenticate (OIDC flow)
    IdP->>C: id_token (JWT)
    C->>P: POST /oauth/token
    Note right of C: grant_type=urn:ietf:params:oauth:<br/>token-type:jwt-bearer<br/>subject_token=<id_token><br/>scope=read:logs-*
    P->>P: Validate id_token signature (JWKS from IdP)
    P->>P: Check issuer against configured providers
    P->>P: Validate audience
    P->>P: Check expiry
    P->>P: Map external scopes → oauth4os scopes
    P->>P: Issue oauth4os access_token
    P->>C: 200 {access_token, token_type: "Bearer"}
```

**Config:**
```yaml
providers:
  - name: keycloak
    issuer: https://keycloak.example.com/realms/opensearch
    audience: ["oauth4os"]
    jwks_uri: auto  # OIDC auto-discovery
```

## 5. Token Introspection (RFC 7662)

Resource servers or admin tools query token validity.

```mermaid
sequenceDiagram
    participant RS as Resource Server
    participant P as oauth4os Proxy

    RS->>P: POST /oauth/introspect
    Note right of RS: token=tok_abc123
    P->>P: Lookup token
    alt Token valid
        P->>RS: 200 {active: true, scope, client_id, exp, iat}
    else Token revoked/expired/unknown
        P->>RS: 200 {active: false}
    end
```

**Response (active):**
```json
{
  "active": true,
  "scope": "read:logs-* write:dashboards",
  "client_id": "my-agent",
  "sub": "my-agent",
  "exp": 1712890800,
  "iat": 1712887200,
  "token_type": "Bearer"
}
```

No token details are leaked for inactive tokens — only `{"active": false}`.

## 6. Dynamic Client Registration (RFC 7591)

Clients self-register via API. Scope allowlist prevents privilege escalation.

```mermaid
sequenceDiagram
    participant C as New Client
    participant P as oauth4os Proxy

    C->>P: POST /oauth/register
    Note right of C: {client_name: "my-bot",<br/>redirect_uris: ["http://localhost/cb"],<br/>scope: "read:logs-*"}
    P->>P: Validate scopes against allowlist
    P->>P: Generate client_id + client_secret
    P->>P: Register with token manager
    P->>C: 201 {client_id, client_secret, scope, grant_types}
    C->>P: POST /oauth/token (use new credentials)
```

**Retrieve client (secret hidden):**
```bash
curl http://localhost:8443/oauth/register/client_abc123
# Returns metadata without client_secret
```

## 7. Device Authorization (RFC 8628)

For CLI tools and IoT devices without a browser. User authorizes on a separate device.

```
CLI Device                  oauth4os                    User's Browser
    │                          │                             │
    │  POST /oauth/device/code │                             │
    │  client_id=cli-tool      │                             │
    │ ────────────────────────▶│                             │
    │                          │                             │
    │  {device_code, user_code,│                             │
    │   verification_uri,      │                             │
    │   interval: 5}           │                             │
    │◀─────────────────────────│                             │
    │                          │                             │
    │  Display to user:        │                             │
    │  "Go to /oauth/device    │                             │
    │   Enter code: ABCD-1234" │                             │
    │                          │     GET /oauth/device       │
    │                          │◀────────────────────────────│
    │                          │     Enter code + approve    │
    │                          │◀────────────────────────────│
    │                          │                             │
    │  POST /oauth/device/token│                             │
    │  (poll every 5s)         │                             │
    │ ────────────────────────▶│                             │
    │                          │  Before approve:            │
    │  {error:                 │  authorization_pending      │
    │   authorization_pending} │                             │
    │◀─────────────────────────│                             │
    │                          │  After approve:             │
    │  POST /oauth/device/token│                             │
    │ ────────────────────────▶│                             │
    │  {access_token, ...}     │                             │
    │◀─────────────────────────│                             │
```

```bash
# Request device code
curl -X POST https://proxy:8443/oauth/device/code \
  -d "client_id=cli-tool&scope=read:logs-*"

# Poll for token (repeat until success)
curl -X POST https://proxy:8443/oauth/device/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=<code>"
```

## 8. API Key Authentication

Stateless authentication via `X-API-Key` header. No OAuth flow — keys are pre-provisioned.

```
Client                      oauth4os                    OpenSearch
    │                          │                             │
    │  GET /logs/_search       │                             │
    │  X-API-Key: osk_abc123   │                             │
    │ ────────────────────────▶│                             │
    │                          │  Validate key prefix + hash │
    │                          │  Check rate limit (per key) │
    │                          │  Map key scopes → roles     │
    │                          │  Cedar evaluation           │
    │                          │                             │
    │                          │  Forward with roles         │
    │                          │ ────────────────────────────▶│
    │  200 {hits: [...]}       │                             │
    │◀─────────────────────────│◀────────────────────────────│
```

```bash
# Create an API key
curl -X POST https://proxy:8443/admin/apikeys \
  -H "Content-Type: application/json" \
  -d '{"client_id":"my-agent","name":"ci-key","scopes":["read:logs-*"]}'

# Use it
curl -H "X-API-Key: osk_abc123..." \
  https://proxy:8443/logs-demo/_search
```

## 9. Pushed Authorization Requests (RFC 9126)

Client pushes auth params server-side before redirecting the user — prevents parameter tampering.

```mermaid
sequenceDiagram
    participant C as Client
    participant P as oauth4os Proxy
    participant U as User Browser

    C->>P: POST /oauth/par (client_id, scope, redirect_uri, code_challenge)
    P-->>C: 201 {request_uri, expires_in: 60}
    C->>U: Redirect to /oauth/authorize?request_uri=urn:...
    U->>P: GET /oauth/authorize?request_uri=urn:...
    P->>P: Resolve request_uri (one-time use)
    P-->>U: Consent screen
    U->>P: POST /oauth/consent (approve)
    P-->>U: Redirect to redirect_uri?code=...
```

## 10. Client Initiated Backchannel Authentication (CIBA)

Backend services authenticate users without browser redirects — user approves on a separate device.

```mermaid
sequenceDiagram
    participant S as Backend Service
    participant P as oauth4os Proxy
    participant U as User (separate device)

    S->>P: POST /oauth/bc-authorize (client_id, login_hint, scope)
    P-->>S: 200 {auth_req_id, expires_in: 300}
    P->>U: Notification (approval page)
    U->>P: POST /oauth/bc-approve (auth_req_id, action=approve)
    S->>P: POST /oauth/bc-token (auth_req_id) [poll]
    P-->>S: 400 {error: authorization_pending}
    S->>P: POST /oauth/bc-token (auth_req_id) [poll again]
    P-->>S: 200 {access_token, refresh_token}
```

## Flow Selection Guide

| Use Case | Flow | Why |
|---|---|---|
| AI agent / bot | Client Credentials | No user interaction needed |
| CI/CD pipeline | Client Credentials or API Key | Automated, scoped access |
| Browser SPA | PKCE | No client secret in browser |
| CLI tool (interactive) | PKCE or Device Flow | Interactive login, secure |
| CLI tool (headless) | Device Flow | No local browser needed |
| IoT device | Device Flow | Authorize on separate device |
| External IdP federation | Token Exchange | Reuse existing OIDC tokens |
| Long-running service | Client Credentials + Refresh | Auto-rotate without re-auth |
| Admin monitoring | Introspection | Check token validity |
| Self-service onboarding | Registration | Automated client provisioning |
| Simple scripts | API Key | No OAuth flow, pre-provisioned |

## Security Summary

| Flow | Key Protection |
|---|---|
| Client Credentials | Constant-time secret compare |
| PKCE | S256 challenge, redirect_uri allowlist, one-time codes |
| Refresh | Token rotation, reuse detection, family revocation |
| Token Exchange | JWKS signature verification, issuer/audience validation |
| Introspection | No details leaked for inactive tokens |
| Registration | Scope allowlist, redirect_uri binding |
| Device Flow | Short-lived user codes, 10-min expiry, one-time use |
| API Key | Hashed storage, prefix-based lookup, per-key rate limits |
| PAR | Server-side param storage, one-time URI, 60s expiry |
| CIBA | Backchannel auth, 5-min expiry, one-time token issuance |
