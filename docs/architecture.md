# Architecture — oauth4os

oauth4os is a reverse proxy that adds OAuth 2.0 token management to OpenSearch with zero changes to OpenSearch itself. This document covers the internal architecture, request lifecycle, and key subsystems.

## System Overview

```
                          ┌─────────────────────────────────────────────┐
                          │              oauth4os proxy (:8443)         │
                          │                                             │
 ┌──────────┐   HTTPS     │  ┌─────────┐  ┌─────────┐  ┌───────────┐  │
 │ AI Agent │────────────▶│  │ Tracing │─▶│  Rate   │─▶│    JWT    │  │
 │ CLI      │             │  │         │  │ Limiter │  │ Validator │  │
 │ CI/CD    │             │  └─────────┘  └─────────┘  └─────┬─────┘  │
 │ Browser  │             │                                   │        │
 │ MCP      │             │  ┌─────────┐  ┌─────────┐  ┌─────▼─────┐  │
 └──────────┘             │  │  Audit  │◀─│  Cedar  │◀─│  Scope    │  │
                          │  │   Log   │  │ Engine  │  │  Mapper   │  │
                          │  └────┬────┘  └─────────┘  └───────────┘  │
                          │       │                                    │
                          │  ┌────▼──────────────────────────────┐     │
                          │  │        Reverse Proxy               │     │
                          │  │  ┌──────────┐  ┌───────────────┐  │     │
                          │  │  │ Direct   │  │ SigV4 Signer  │  │     │
                          │  │  │ Forward  │  │ (for AOSS)    │  │     │
                          │  │  └────┬─────┘  └──────┬────────┘  │     │
                          │  └───────┼───────────────┼───────────┘     │
                          └──────────┼───────────────┼─────────────────┘
                                     │               │
                          ┌──────────▼───┐  ┌────────▼──────────┐
                          │  OpenSearch  │  │  OpenSearch        │
                          │  Engine      │  │  Serverless (AOSS) │
                          │  :9200       │  │  (AWS SigV4)       │
                          └──────────────┘  └───────────────────┘
```

## Internal Packages

| Package | Responsibility |
|---------|---------------|
| `jwt` | JWT validation, JWKS cache with auto-refresh |
| `scope` | Map OAuth scopes to OpenSearch backend roles |
| `cedar` | Cedar policy evaluation, multi-tenant |
| `token` | Token lifecycle — issue, refresh, revoke, reuse detection |
| `pkce` | PKCE authorization code flow + consent screen |
| `introspect` | RFC 7662 token introspection |
| `exchange` | RFC 8693 token exchange |
| `registration` | RFC 7591 dynamic client registration |
| `ratelimit` | Per-client token bucket rate limiter |
| `tracing` | OpenTelemetry-style distributed tracing |
| `audit` | Structured JSON audit logging |
| `admin` | REST API for live config changes |
| `analytics` | Per-client, per-scope request metrics |
| `keyring` | RSA key rotation + JWKS endpoint |
| `discovery` | OIDC Discovery (/.well-known/openid-configuration) |
| `config` | YAML config loader |
| `federation` | Multi-cluster routing by index pattern |
| `sigv4` | AWS SigV4 request signing for AOSS |
| `ipfilter` | Per-client IP allowlist/denylist |
| `mtls` | Mutual TLS client certificate auth |
| `session` | Session tracking, revocation, force logout |
| `logging` | Structured leveled logging |
| `backup` | Config backup/restore bundles |
| `webhook` | External webhook notifications |
| `demo` | Demo web app backend |

---

## Request Lifecycle

Every proxied request passes through a fixed middleware chain. Each stage is a separate span in the trace.

```
 Client Request
       │
       ▼
 ┌─────────────────────────────────────────────────────────┐
 │ 1. TRACING                                              │
 │    Assign X-Request-ID, start root span                 │
 │    Record: method, path, client IP                      │
 └──────────────────────┬──────────────────────────────────┘
                        │
 ┌──────────────────────▼──────────────────────────────────┐
 │ 2. IP FILTER                                            │
 │    Check client IP against allowlist/denylist            │
 │    → 403 if denied                                      │
 └──────────────────────┬──────────────────────────────────┘
                        │
 ┌──────────────────────▼──────────────────────────────────┐
 │ 3. mTLS (optional)                                      │
 │    Extract client identity from TLS certificate          │
 │    CN → DNS SANs → Email SANs                           │
 └──────────────────────┬──────────────────────────────────┘
                        │
 ┌──────────────────────▼──────────────────────────────────┐
 │ 4. RATE LIMITER                                         │
 │    Token bucket per client_id                           │
 │    Scope-aware RPM limits                               │
 │    → 429 + Retry-After header if exceeded               │
 └──────────────────────┬──────────────────────────────────┘
                        │
 ┌──────────────────────▼──────────────────────────────────┐
 │ 5. JWT VALIDATION                                       │
 │    Extract Bearer token from Authorization header       │
 │    Validate signature (RS256/ES256) via cached JWKS     │
 │    Check exp, iss, aud claims                           │
 │    Extract client_id and scopes                         │
 │    → 401 if invalid/expired                             │
 └──────────────────────┬──────────────────────────────────┘
                        │
 ┌──────────────────────▼──────────────────────────────────┐
 │ 6. SCOPE MAPPING                                        │
 │    Map OAuth scopes → OpenSearch backend_roles           │
 │    Tenant-aware: per-issuer mappings with global fallback│
 │    Set X-Proxy-User and X-Proxy-Roles headers           │
 └──────────────────────┬──────────────────────────────────┘
                        │
 ┌──────────────────────▼──────────────────────────────────┐
 │ 7. CEDAR POLICY EVALUATION                              │
 │    Evaluate permit/forbid rules against:                │
 │      principal = client_id                              │
 │      action    = HTTP method                            │
 │      resource  = index name from URL path               │
 │    Deny-overrides: any forbid → 403                     │
 │    → 403 if denied                                      │
 └──────────────────────┬──────────────────────────────────┘
                        │
 ┌──────────────────────▼──────────────────────────────────┐
 │ 8. AUDIT LOG                                            │
 │    Record: timestamp, client_id, method, path,          │
 │    scopes, roles, source IP, decision                   │
 └──────────────────────┬──────────────────────────────────┘
                        │
 ┌──────────────────────▼──────────────────────────────────┐
 │ 9. REVERSE PROXY                                        │
 │    Route to upstream (federation or single cluster)     │
 │    If AOSS: sign with SigV4                             │
 │    Forward response to client                           │
 │    Record: status code, duration                        │
 └─────────────────────────────────────────────────────────┘
```

**Latency budget**: The middleware chain adds ~1-3ms per request (JWT cache hit). JWKS refresh happens asynchronously in the background.

---

## PKCE Authorization Flow

Browser clients use PKCE (RFC 7636) with a consent screen.

```
 Browser                    oauth4os                    OIDC Provider
    │                          │                             │
    │  1. Generate verifier    │                             │
    │     + S256 challenge     │                             │
    │                          │                             │
    │  2. GET /oauth/authorize │                             │
    │     ?client_id=app       │                             │
    │     &code_challenge=xxx  │                             │
    │     &redirect_uri=...    │                             │
    │     &scope=read:logs-*   │                             │
    │ ────────────────────────▶│                             │
    │                          │                             │
    │  3. Consent screen       │                             │
    │◀─────────────────────────│                             │
    │                          │                             │
    │  ┌─────────────────────────────────────────┐           │
    │  │  🔐 oauth4os                            │           │
    │  │                                         │           │
    │  │  "app" requests access:                 │           │
    │  │                                         │           │
    │  │  👁  read:logs-*                        │           │
    │  │     Read data from indices (logs-*)     │           │
    │  │                                         │           │
    │  │  [Deny]              [Approve]          │           │
    │  └─────────────────────────────────────────┘           │
    │                          │                             │
    │  4. POST /oauth/consent  │                             │
    │     action=approve       │                             │
    │ ────────────────────────▶│                             │
    │                          │  Store auth code            │
    │  5. 302 redirect_uri     │  (10 min TTL, one-time)    │
    │     ?code=abc123         │                             │
    │◀─────────────────────────│                             │
    │                          │                             │
    │  6. POST /oauth/token    │                             │
    │     grant_type=          │                             │
    │       authorization_code │                             │
    │     code=abc123          │                             │
    │     code_verifier=xxx    │                             │
    │ ────────────────────────▶│                             │
    │                          │  Verify:                    │
    │                          │  SHA256(verifier)==challenge │
    │                          │  Code not expired/reused    │
    │                          │                             │
    │  7. {access_token, ...}  │                             │
    │◀─────────────────────────│                             │
    │                          │                             │
    │  8. GET /logs-*/_search  │                             │
    │     Authorization:       │                             │
    │       Bearer <token>     │                             │
    │ ────────────────────────▶│  ──── proxy to OpenSearch   │
```

**Security properties**:
- Code verifier never leaves the browser (not sent to authorize endpoint)
- Auth codes are single-use and expire after 10 minutes
- Consent is per-request — no persistent grants
- `redirect_uri` validated against client's registered allowlist

---

## SigV4 Signing Flow

When the upstream is OpenSearch Serverless (AOSS), the proxy signs requests with AWS SigV4.

```
 Client                     oauth4os                        AOSS
    │                          │                              │
    │  GET /logs-demo/_search  │                              │
    │  Authorization:          │                              │
    │    Bearer <oauth-token>  │                              │
    │ ────────────────────────▶│                              │
    │                          │                              │
    │                          │  1. Validate JWT (normal)    │
    │                          │  2. Map scopes → roles       │
    │                          │  3. Cedar evaluation         │
    │                          │                              │
    │                          │  4. Strip Authorization hdr  │
    │                          │  5. Get AWS credentials      │
    │                          │     (env / instance role)    │
    │                          │  6. Compute SigV4 signature: │
    │                          │     - Canonical request      │
    │                          │     - String to sign         │
    │                          │     - HMAC-SHA256 chain      │
    │                          │  7. Set headers:             │
    │                          │     Authorization: AWS4-...  │
    │                          │     X-Amz-Date: ...          │
    │                          │     X-Amz-Security-Token:... │
    │                          │     Host: <aoss-endpoint>    │
    │                          │                              │
    │                          │  GET /logs-demo/_search      │
    │                          │  (SigV4 signed)              │
    │                          │ ────────────────────────────▶│
    │                          │                              │
    │                          │  200 {hits: [...]}           │
    │                          │◀─────────────────────────────│
    │                          │                              │
    │  200 {hits: [...]}       │                              │
    │◀─────────────────────────│                              │
```

**Key details**:
- Credentials refresh automatically (IAM role or environment variables)
- The `Host` header is set to the AOSS endpoint (required by SigV4)
- Original `Authorization` header (Bearer token) is stripped before signing
- Request body is included in the signature hash
- Clock skew tolerance: ±5 minutes (AWS requirement)

---

## Cedar Policy Evaluation

Cedar policies provide fine-grained authorization beyond scope-to-role mapping.

### Evaluation Model

```
                    ┌──────────────────────────┐
                    │     Cedar Request         │
                    │                           │
                    │  principal: "client-abc"  │
                    │  action:    "POST"        │
                    │  resource:  "logs-demo"   │
                    │  context:                 │
                    │    scopes: [read:logs-*]  │
                    │    ip: 10.0.1.50          │
                    │    tenant: "keycloak"     │
                    └────────────┬──────────────┘
                                 │
              ┌──────────────────▼──────────────────┐
              │         Policy Evaluation            │
              │                                      │
              │  1. Load tenant policies (by issuer) │
              │  2. Load global policies             │
              │  3. Evaluate all matching policies   │
              │                                      │
              │  ┌────────────────────────────────┐  │
              │  │ permit(                        │  │
              │  │   principal,                   │  │
              │  │   action in ["GET","POST"],    │  │
              │  │   resource                     │  │
              │  │ ) when {                       │  │
              │  │   context.scopes.contains(     │  │
              │  │     "read:" + resource)        │  │
              │  │ };                             │  │
              │  └────────────────────────────────┘  │
              │                                      │
              │  ┌────────────────────────────────┐  │
              │  │ forbid(                        │  │
              │  │   principal,                   │  │
              │  │   action,                      │  │
              │  │   resource == ".opendistro_*"  │  │
              │  │ );                             │  │
              │  └────────────────────────────────┘  │
              │                                      │
              └──────────────────┬───────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │  Decision Algorithm:     │
                    │                          │
                    │  Any FORBID → DENY       │
                    │  No PERMIT  → DENY       │
                    │  Otherwise  → ALLOW      │
                    │                          │
                    │  (deny-overrides model)  │
                    └──────────────────────────┘
```

**Multi-tenant**: Each OIDC issuer can have its own policy set. Global policies apply to all tenants. A Keycloak realm and a Dex instance can have completely different authorization rules.

**Admin API**: Policies are managed via REST:
- `GET /admin/policies` — list all policies
- `POST /admin/policies` — add a policy
- `DELETE /admin/policies/{id}` — remove a policy

---

## Token Lifecycle

```
 ┌──────────────────────────────────────────────────────────────────┐
 │                        Token States                              │
 │                                                                  │
 │  ┌────────┐   issue    ┌────────┐   expire    ┌─────────┐      │
 │  │        │───────────▶│        │────────────▶│         │      │
 │  │  None  │            │ Active │             │ Expired │      │
 │  │        │            │        │             │         │      │
 │  └────────┘            └───┬────┘             └─────────┘      │
 │                            │                                    │
 │                   refresh  │  revoke                            │
 │                   (rotate) │                                    │
 │                            │                                    │
 │                    ┌───────▼───────┐                            │
 │                    │               │                            │
 │                    │   Revoked     │                            │
 │                    │               │                            │
 │                    └───────────────┘                            │
 │                                                                  │
 └──────────────────────────────────────────────────────────────────┘
```

### Grant Types

| Grant | Flow | Use Case |
|-------|------|----------|
| `client_credentials` | Client sends ID + secret → gets access token | Machine-to-machine (CI/CD, agents) |
| `authorization_code` | Browser PKCE flow → consent → code → token | Browser apps, interactive users |
| `refresh_token` | Exchange refresh token → new access + refresh | Extend sessions without re-auth |
| `urn:ietf:params:oauth:grant-type:token-exchange` | Swap external IdP JWT → scoped proxy token | Federation, cross-IdP access |

### Refresh Token Rotation

```
 Client                     oauth4os
    │                          │
    │  POST /oauth/token       │
    │  grant_type=refresh_token│
    │  refresh_token=RT-1      │
    │ ────────────────────────▶│
    │                          │  1. Validate RT-1
    │                          │  2. Issue new AT-2 + RT-2
    │                          │  3. Invalidate RT-1
    │                          │  4. Link RT-2 → RT-1 (family)
    │  {access_token: AT-2,    │
    │   refresh_token: RT-2}   │
    │◀─────────────────────────│
    │                          │
    │  (attacker replays RT-1) │
    │  POST /oauth/token       │
    │  refresh_token=RT-1      │
    │ ────────────────────────▶│
    │                          │  RT-1 already used!
    │                          │  → Revoke entire family
    │                          │  → RT-2 also invalidated
    │  401 invalid_grant       │
    │◀─────────────────────────│
```

**Reuse detection**: If a refresh token is used twice, the entire token family is revoked. This protects against token theft — if an attacker captures a refresh token, the legitimate client's next refresh will trigger revocation of all tokens in the family.

---

## Multi-Cluster Federation

The federation router directs requests to different OpenSearch clusters based on index patterns.

```
                          ┌──────────────────────┐
                          │   Federation Router   │
                          │                       │
  GET /logs-app-a/_search │  Rules:               │
 ────────────────────────▶│  logs-app-a → Cluster A│──▶ Cluster A (us-east-1)
                          │  logs-app-b → Cluster B│
  GET /logs-app-b/_search │  metrics-*  → Cluster C│
 ────────────────────────▶│  *          → Default  │──▶ Cluster B (us-west-2)
                          │                       │
  GET /_cluster/health    │                       │
 ────────────────────────▶│  (no index prefix)    │──▶ Default cluster
                          └──────────────────────┘
```

Configuration:

```yaml
clusters:
  - name: app-a
    url: https://cluster-a.example.com:9200
    indices: ["logs-app-a", "logs-app-a-*"]
  - name: app-b
    url: https://cluster-b.example.com:9200
    indices: ["logs-app-b*"]
  - name: default
    url: https://default.example.com:9200
    indices: ["*"]
```

Pattern matching supports `*` wildcards. The first matching cluster wins. Requests without an index prefix (e.g., `/_cluster/health`) go to the default cluster.

---

## Metrics

The proxy exposes Prometheus metrics at `GET /metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `oauth4os_requests_total` | Counter | Total proxied requests |
| `oauth4os_requests_active` | Gauge | In-flight requests |
| `oauth4os_requests_failed` | Counter | Failed requests (4xx/5xx) |
| `oauth4os_auth_success` | Counter | Successful JWT validations |
| `oauth4os_auth_failed` | Counter | Failed JWT validations |
| `oauth4os_cedar_denied` | Counter | Cedar policy denials |
| `oauth4os_rate_limited` | Counter | Rate limit rejections |
| `oauth4os_upstream_errors` | Counter | Upstream connection errors |
| `oauth4os_uptime_seconds` | Gauge | Proxy uptime |

---

## Configuration Loading

```
 Startup
    │
    ▼
 Load config.yaml
    │
    ├─▶ upstream.engine / upstream.dashboards
    ├─▶ providers[] → JWKS URLs → background refresh
    ├─▶ scope_mapping → in-memory lookup table
    ├─▶ rate_limits → per-client token buckets
    ├─▶ ip_filter → CIDR parsers per client
    ├─▶ mtls → load CA certificate pool
    ├─▶ clusters[] → federation router
    ├─▶ sigv4 → AWS credential chain
    └─▶ listen address + TLS config
         │
         ▼
    Start HTTP(S) server
    Register graceful shutdown (SIGTERM/SIGINT)
```

All configuration is hot-reloadable via the Admin API (`/admin/*`) without restarting the proxy.

---

## Dependencies

oauth4os uses only 2 external Go modules:

| Module | Purpose |
|--------|---------|
| `github.com/golang-jwt/jwt/v5` | JWT parsing and validation |
| `gopkg.in/yaml.v3` | YAML config file parsing |

Everything else — HTTP server, TLS, crypto, rate limiting, tracing, reverse proxy — uses Go's standard library. This minimizes supply chain risk and keeps the binary under 15MB.
