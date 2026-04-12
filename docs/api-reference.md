# API Reference

Complete endpoint reference for oauth4os. All examples use the live demo at `https://f5cmk2hxwx.us-west-2.awsapprunner.com`. Replace with your proxy URL.

---

## Token Endpoints

### POST /oauth/token

Issue an access token. Supports three grant types.

**Client Credentials** (machine-to-machine):

```bash
curl -X POST https://proxy:8443/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=my-agent" \
  -d "client_secret=secret" \
  -d "scope=read:logs-*"
```

Response:
```json
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "rt-abc123...",
  "scope": "read:logs-*"
}
```

**Refresh Token**:

```bash
curl -X POST https://proxy:8443/oauth/token \
  -d "grant_type=refresh_token" \
  -d "refresh_token=rt-abc123..."
```

**Authorization Code** (PKCE):

```bash
curl -X POST https://proxy:8443/oauth/token \
  -d "grant_type=authorization_code" \
  -d "code=abc123" \
  -d "code_verifier=original-verifier" \
  -d "redirect_uri=http://localhost:3000/callback"
```

| Parameter | Required | Description |
|-----------|----------|-------------|
| `grant_type` | Yes | `client_credentials`, `refresh_token`, or `authorization_code` |
| `client_id` | Yes (credentials) | Client identifier |
| `client_secret` | Yes (credentials) | Client secret |
| `scope` | No | Space-separated scopes (e.g. `read:logs-* admin`) |
| `refresh_token` | Yes (refresh) | Refresh token to exchange |
| `code` | Yes (auth code) | Authorization code from PKCE flow |
| `code_verifier` | Yes (auth code) | PKCE code verifier |
| `redirect_uri` | Yes (auth code) | Must match the authorize request |

---

### GET /oauth/tokens

List all active tokens.

```bash
curl https://proxy:8443/oauth/tokens \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Response:
```json
[
  {
    "id": "tok-abc123",
    "client_id": "my-agent",
    "scopes": ["read:logs-*"],
    "created_at": "2025-04-12T06:00:00Z",
    "expires_at": "2025-04-12T07:00:00Z",
    "active": true
  }
]
```

---

### GET /oauth/token/{id}

Get details for a specific token.

```bash
curl https://proxy:8443/oauth/token/tok-abc123 \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

### DELETE /oauth/token/{id}

Revoke a token immediately.

```bash
curl -X DELETE https://proxy:8443/oauth/token/tok-abc123 \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Response: `204 No Content`

---

### POST /oauth/introspect

Token introspection (RFC 7662). Check if a token is active.

```bash
curl -X POST https://proxy:8443/oauth/introspect \
  -d "token=eyJhbG..."
```

Response (active):
```json
{
  "active": true,
  "client_id": "my-agent",
  "scope": "read:logs-*",
  "exp": 1712905200,
  "iat": 1712901600
}
```

Response (inactive):
```json
{"active": false}
```

---

## PKCE / Authorization

### GET /oauth/authorize

Start a PKCE authorization code flow. Renders the consent screen.

```bash
# Browser redirect — not typically called via curl
https://proxy:8443/oauth/authorize?\
  response_type=code&\
  client_id=my-app&\
  redirect_uri=http://localhost:3000/callback&\
  code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&\
  code_challenge_method=S256&\
  scope=read:logs-*&\
  state=random-state
```

| Parameter | Required | Description |
|-----------|----------|-------------|
| `client_id` | Yes | Registered client ID |
| `redirect_uri` | Yes | Must match registered URI |
| `code_challenge` | Yes | Base64url(SHA256(code_verifier)) |
| `code_challenge_method` | No | Only `S256` supported (default) |
| `scope` | No | Requested scopes |
| `state` | No | Opaque value returned in callback |

Returns: HTML consent screen. On approve, redirects to `redirect_uri?code=xxx&state=yyy`.

---

### POST /oauth/consent

Submit consent decision (called by the consent screen form).

| Parameter | Required | Description |
|-----------|----------|-------------|
| `consent_id` | Yes | From the consent form hidden field |
| `action` | Yes | `approve` or `deny` |

On approve: `302` redirect to `redirect_uri?code=xxx&state=yyy`
On deny: `302` redirect to `redirect_uri?error=access_denied&state=yyy`

---

## Client Registration (RFC 7591)

### POST /oauth/register

Register a new OAuth client.

```bash
curl -X POST https://proxy:8443/oauth/register \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "my-agent",
    "scope": "read:logs-* write:logs-*",
    "redirect_uris": ["http://localhost:3000/callback"]
  }'
```

Response:
```json
{
  "client_id": "abc123",
  "client_secret": "sec-xyz...",
  "client_name": "my-agent",
  "scope": "read:logs-* write:logs-*",
  "redirect_uris": ["http://localhost:3000/callback"],
  "created_at": "2025-04-12T06:00:00Z"
}
```

---

### GET /oauth/register

List all registered clients.

```bash
curl https://proxy:8443/oauth/register
```

---

### GET /oauth/register/{client_id}

Get a specific client's details.

```bash
curl https://proxy:8443/oauth/register/abc123
```

---

### PUT /oauth/register/{client_id}

Update a client.

```bash
curl -X PUT https://proxy:8443/oauth/register/abc123 \
  -H "Content-Type: application/json" \
  -d '{"client_name": "renamed-agent", "scope": "read:logs-*"}'
```

---

### DELETE /oauth/register/{client_id}

Delete a client.

```bash
curl -X DELETE https://proxy:8443/oauth/register/abc123
```

---

### POST /oauth/register/{client_id}/rotate

Rotate a client's secret. Returns the new secret; the old one is immediately invalidated.

```bash
curl -X POST https://proxy:8443/oauth/register/abc123/rotate
```

Response:
```json
{
  "client_id": "abc123",
  "client_secret": "sec-new..."
}
```

---

## Admin Endpoints

### GET /admin/analytics

Token usage analytics — top clients, scope distribution, top indices.

```bash
curl https://proxy:8443/admin/analytics
```

Response:
```json
{
  "top_clients": [
    {"client_id": "my-agent", "requests": 1523, "last_seen": "2025-04-12T06:55:00Z"}
  ],
  "scope_distribution": [
    {"name": "read:logs-*", "count": 1200}
  ],
  "top_indices": [
    {"name": "logs-demo", "count": 980}
  ]
}
```

---

### GET /admin/audit

Query the audit log.

```bash
# Last 10 entries
curl "https://proxy:8443/admin/audit?limit=10"

# Filter by client
curl "https://proxy:8443/admin/audit?client_id=my-agent&limit=50"
```

| Parameter | Description |
|-----------|-------------|
| `limit` | Max entries to return (default 100) |
| `client_id` | Filter by client ID |

---

### GET /admin/clusters

Multi-cluster federation status.

```bash
curl https://proxy:8443/admin/clusters
```

Response:
```json
{
  "clusters": ["app-a", "app-b", "default"],
  "active": 3
}
```

---

### GET /admin/sessions

List active sessions.

```bash
curl https://proxy:8443/admin/sessions
```

---

### DELETE /admin/sessions/{id}

Revoke a specific session.

```bash
curl -X DELETE https://proxy:8443/admin/sessions/sess-abc123
```

---

### POST /admin/sessions/logout

Force logout all sessions for a client.

```bash
curl -X POST https://proxy:8443/admin/sessions/logout \
  -H "Content-Type: application/json" \
  -d '{"client_id": "my-agent"}'
```

---

### POST /admin/apikeys

Create an API key.

```bash
curl -X POST https://proxy:8443/admin/apikeys \
  -H "Content-Type: application/json" \
  -d '{"client_id": "my-agent", "name": "ci-key", "scopes": ["read:logs-*"]}'
```

---

### GET /admin/apikeys/{client_id}

List API keys for a client.

```bash
curl https://proxy:8443/admin/apikeys/my-agent
```

---

### DELETE /admin/apikeys/{id}

Delete an API key.

```bash
curl -X DELETE https://proxy:8443/admin/apikeys/key-abc123
```

---

### GET /admin/health

Admin health dashboard (detailed system status).

```bash
curl https://proxy:8443/admin/health
```

---

### GET /admin/clients

List all registered clients (secrets excluded).

```bash
curl https://proxy:8443/admin/clients
```

Response:
```json
[
  {"client_id": "abc123", "client_name": "my-app", "scope": "read:logs-*"},
  {"client_id": "def456", "client_name": "demo-cli", "scope": "read:logs"}
]
```

---

### GET /admin/tokens

List active tokens (values excluded).

```bash
curl https://proxy:8443/admin/tokens
```

Response:
```json
[
  {"token_id": "tok_1", "client_id": "abc123", "scope": "read:logs-*", "expires_at": "2026-04-13T10:00:00Z"},
  {"token_id": "tok_2", "client_id": "def456", "scope": "read:logs", "expires_at": "2026-04-13T11:00:00Z"}
]
```

---

### POST /admin/keys/rotate

Trigger immediate signing key rotation. Previous key stays in JWKS.

```bash
curl -X POST https://proxy:8443/admin/keys/rotate
```

Response:
```json
{"status": "rotated", "kid": "new-key-id"}
```

---

## Discovery Endpoints

### GET /.well-known/openid-configuration

OIDC Discovery document.

```bash
curl https://proxy:8443/.well-known/openid-configuration
```

Response:
```json
{
  "issuer": "https://proxy:8443",
  "authorization_endpoint": "https://proxy:8443/oauth/authorize",
  "token_endpoint": "https://proxy:8443/oauth/token",
  "introspection_endpoint": "https://proxy:8443/oauth/introspect",
  "jwks_uri": "https://proxy:8443/.well-known/jwks.json",
  "registration_endpoint": "https://proxy:8443/oauth/register",
  "scopes_supported": ["read:logs-*", "write:logs-*", "admin"],
  "response_types_supported": ["code"],
  "grant_types_supported": [
    "client_credentials",
    "authorization_code",
    "refresh_token",
    "urn:ietf:params:oauth:grant-type:token-exchange"
  ],
  "token_endpoint_auth_methods_supported": ["client_secret_post"],
  "code_challenge_methods_supported": ["S256"]
}
```

---

### GET /.well-known/jwks.json

JSON Web Key Set for token verification.

```bash
curl https://proxy:8443/.well-known/jwks.json
```

Response:
```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "key-1",
      "use": "sig",
      "alg": "RS256",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

---

## Health & Metrics

### GET /health

Basic health check. Use for load balancer probes.

```bash
curl https://proxy:8443/health
```

Response:
```json
{"status": "ok", "version": "1.0.0", "uptime": "2h15m"}
```

---

### GET /health/deep

Deep health check — verifies upstream connectivity, JWKS freshness, TLS cert expiry.

```bash
curl https://proxy:8443/health/deep
```

Response:
```json
{
  "status": "ok",
  "upstream": {"status": "ok", "latency_ms": 45},
  "jwks": {"status": "ok", "keys": 1, "last_refresh": "2025-04-12T06:50:00Z"},
  "tls": {"status": "ok", "expires": "2026-01-01T00:00:00Z"}
}
```

---

### GET /metrics

Prometheus-format metrics.

```bash
curl https://proxy:8443/metrics
```

Response:
```
oauth4os_requests_total 15234
oauth4os_requests_active 3
oauth4os_requests_failed 42
oauth4os_auth_success 15192
oauth4os_auth_failed 42
oauth4os_cedar_denied 7
oauth4os_rate_limited 12
oauth4os_upstream_errors 3
oauth4os_uptime_seconds 8100
```

---

## Developer Endpoints

### GET /developer/docs

Interactive API documentation page (HTML).

### GET /developer/openapi.yaml

OpenAPI 3.0 specification in YAML format.

```bash
curl https://proxy:8443/developer/openapi.yaml
```

### GET /developer/analytics

Developer analytics dashboard (HTML) — top clients, scope distribution, request timeline.

---

### GET /version

Machine-readable version info.

```bash
curl https://proxy:8443/version
```

Response:
```json
{"version": "1.0.0", "go": "go1.22", "os": "linux", "arch": "amd64"}
```

---

### GET /ready

Kubernetes readiness probe. Returns 503 during graceful shutdown.

```bash
curl https://proxy:8443/ready
```

Response (ready): `{"status":"ready"}`
Response (shutting down): `503 {"status":"shutting_down"}`

---

### GET /playground

Interactive OAuth playground — step through PKCE flow visually.

### GET /analytics

Token analytics dashboard (HTML) — top clients, scope distribution, request timeline. Auto-refreshes.

---

## Demo & Install

### GET /demo

Demo log viewer web app. Login with PKCE, search logs, scope enforcement demo.

### GET /install.sh

CLI installer script. Detects OS/arch, downloads the CLI wrapper.

```bash
curl -sL https://proxy:8443/install.sh | bash
```

### GET /scripts/oauth4os-demo

Raw CLI demo script (downloaded by install.sh).

---

## Proxy Pass-Through

### ANY /*

All other requests are authenticated and proxied to OpenSearch.

```bash
# Search
curl -H "Authorization: Bearer $TOKEN" \
  https://proxy:8443/logs-demo/_search \
  -H "Content-Type: application/json" \
  -d '{"query":{"match":{"level":"ERROR"}},"size":10}'

# Get cluster health
curl -H "Authorization: Bearer $TOKEN" \
  https://proxy:8443/_cluster/health

# Create an index
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  https://proxy:8443/my-index \
  -H "Content-Type: application/json" \
  -d '{"settings":{"number_of_shards":1}}'

# Bulk index
curl -H "Authorization: Bearer $TOKEN" \
  https://proxy:8443/_bulk \
  -H "Content-Type: application/x-ndjson" \
  --data-binary @data.ndjson
```

**Headers added by proxy**:
- `X-Proxy-User` — client ID from the token
- `X-Proxy-Roles` — OpenSearch backend roles mapped from scopes
- `X-Request-ID` — unique request identifier for tracing

**Headers returned**:
- `X-Request-ID` — same ID for correlation
- `X-RateLimit-Remaining` — requests remaining in current window
- `X-RateLimit-Reset` — seconds until rate limit resets

---

## Error Responses

All errors follow OAuth 2.0 error format:

```json
{
  "error": "error_code",
  "error_description": "Human-readable description"
}
```

| Status | Error | Description |
|--------|-------|-------------|
| 400 | `invalid_request` | Missing or invalid parameters |
| 400 | `invalid_grant` | Auth code expired, reused, or verifier mismatch |
| 401 | `invalid_token` | Token expired, revoked, or signature invalid |
| 403 | `access_denied` | Cedar policy denied the request |
| 403 | `insufficient_scope` | Token lacks required scope |
| 429 | `rate_limited` | Rate limit exceeded (check `Retry-After` header) |
| 502 | `upstream_error` | OpenSearch returned an error |
