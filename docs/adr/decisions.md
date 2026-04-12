# ADR-001: Reverse Proxy Architecture

**Status:** Accepted  
**Date:** 2026-04-12

## Context
OpenSearch needs OAuth 2.0 support without modifying its core. Options: (a) OpenSearch plugin, (b) reverse proxy, (c) API gateway sidecar.

## Decision
Reverse proxy in Go. All traffic flows through oauth4os before reaching OpenSearch.

## Rationale
- Zero changes to OpenSearch — works with any version
- Single binary, no JVM dependency
- Can protect both Engine and Dashboards from one process
- Decoupled upgrade cycle from OpenSearch releases

## Consequences
- Additional network hop (~1ms overhead measured)
- Must handle all HTTP edge cases (WebSocket, chunked transfer, etc.)
- Becomes a single point of failure — requires health checks and redundancy

---

# ADR-002: Forbid-Overrides Policy Model (Cedar)

**Status:** Accepted  
**Date:** 2026-04-12

## Context
Need a policy evaluation model for fine-grained access control. Options: (a) allow-overrides, (b) deny-overrides (forbid-overrides), (c) first-match.

## Decision
Forbid-overrides: any matching `forbid` policy denies the request regardless of `permit` policies.

## Rationale
- Matches AWS Cedar semantics — familiar to AWS users
- Safer default: adding a forbid rule can never be overridden by a permit
- Prevents privilege escalation via permissive policy stacking
- Default deny when no permit matches

## Consequences
- Cannot create "exception" permits that override forbids
- Admin must remove forbid rules to grant access, not add permits

---

# ADR-003: Constant-Time Secret Comparison

**Status:** Accepted  
**Date:** 2026-04-12

## Context
Client authentication compares secrets. Standard `==` is vulnerable to timing attacks.

## Decision
Use `crypto/subtle.ConstantTimeCompare` for all secret comparisons: client secrets, PKCE verifiers.

## Consequences
- Prevents timing-based secret extraction
- Negligible performance impact

---

# ADR-004: Refresh Token Rotation with Reuse Detection

**Status:** Accepted  
**Date:** 2026-04-12

## Context
Refresh tokens are long-lived. If stolen, attacker has persistent access.

## Decision
Rotate on every refresh: old token revoked, new token issued. Track used tokens — if a used token is replayed, revoke the entire token family for that client.

## Rationale
- Limits window of stolen token usability to one refresh cycle
- Reuse detection catches theft even after rotation
- Family revocation forces re-authentication

## Consequences
- Clients must store the new refresh token after every refresh
- Concurrent refresh requests from the same client will trigger reuse detection (by design)

---

# ADR-005: PKCE-Only for Browser Clients

**Status:** Accepted  
**Date:** 2026-04-12

## Context
Browser clients cannot securely store client secrets. Options: (a) implicit flow (deprecated), (b) PKCE, (c) device code flow.

## Decision
PKCE with S256 only. No implicit flow. No plain challenge method.

## Rationale
- OAuth 2.1 deprecates implicit flow
- S256 prevents authorization code interception
- Plain method offers no security benefit

## Consequences
- Older clients that only support implicit flow are not supported
- All browser integrations must implement PKCE

---

# ADR-006: Redirect URI Allowlist

**Status:** Accepted  
**Date:** 2026-04-12

## Context
PKCE flow redirects to client-provided URI with authorization code. Without validation, attacker can steal codes via open redirect.

## Decision
Clients must register redirect URIs at registration time. Exact match only — no wildcards, no pattern matching.

## Rationale
- Prevents open redirect attacks (OWASP Top 10)
- Exact match is the strictest and simplest validation
- Wildcard matching introduces subdomain takeover risks

## Consequences
- Clients must know their redirect URIs at registration time
- localhost URIs must be explicitly registered for development

---

# ADR-007: In-Memory Token Store (MVP)

**Status:** Accepted (temporary)  
**Date:** 2026-04-12

## Context
Token storage options: (a) in-memory, (b) Redis, (c) DynamoDB, (d) SQLite.

## Decision
In-memory for MVP. Documented as known limitation.

## Rationale
- Zero external dependencies for quick start
- Sufficient for development and small deployments
- Interface-based design allows swapping to persistent store

## Consequences
- Tokens lost on restart
- No horizontal scaling (each instance has its own token set)
- Must implement persistent backend before production recommendation

---

# ADR-008: Strip Proxy-Trust Headers

**Status:** Accepted  
**Date:** 2026-04-12

## Context
Proxy sets X-Proxy-User/Roles/Scopes headers for OpenSearch. If client sends these headers on unauthenticated requests, they pass through to upstream — enabling impersonation.

## Decision
Strip X-Proxy-User, X-Proxy-Roles, X-Proxy-Scopes, and Cookie headers on all paths before forwarding to upstream.

## Rationale
- Prevents client impersonation via header injection
- Defense in depth — even if OpenSearch trusts these headers, proxy ensures they're authentic

---

# ADR-009: Generic Error Responses

**Status:** Accepted  
**Date:** 2026-04-12

## Context
Detailed error messages aid debugging but leak internal architecture (file paths, library versions, table names).

## Decision
All client-facing errors return generic messages. Internal details logged server-side only.

## Rationale
- Prevents information disclosure (CWE-209)
- Audit log retains full details for debugging
- Consistent with OAuth 2.0 error response spec (error + error_description only)

---

# ADR-010: OIDC Discovery Issuer Validation

**Status:** Accepted  
**Date:** 2026-04-12

## Context
OIDC auto-discovery fetches `/.well-known/openid-configuration` from the issuer URL. A malicious discovery document could redirect JWKS fetches to an attacker-controlled server (SSRF).

## Decision
Validate that the `issuer` field in the discovery document matches the configured issuer exactly.

## Rationale
- Prevents SSRF via discovery redirect
- Matches OIDC Core spec §3 requirement
- Simple check with high security value

---

# ADR-011: Pushed Authorization Requests (PAR)

**Status:** Accepted  
**Date:** 2026-04-12

## Context
PKCE authorize requests pass parameters in the browser URL. A compromised browser extension or network observer could tamper with redirect_uri, scope, or code_challenge.

## Decision
Implement RFC 9126 PAR. Clients POST auth params to /oauth/par, receive a one-time request_uri, then redirect users with only the opaque URI.

## Rationale
- Auth parameters never appear in browser URL bar
- One-time use prevents replay
- 60s expiry limits window of attack
- Client authentication on the PAR endpoint

## Consequences
- Adds an extra round-trip before authorization
- Clients must implement the PAR flow (not just redirect)

---

# ADR-012: Client Initiated Backchannel Authentication (CIBA)

**Status:** Accepted  
**Date:** 2026-04-12

## Context
Some services (IoT, backend jobs, call centers) need to authenticate users without a browser redirect on the requesting device.

## Decision
Implement CIBA. Service requests auth via backchannel, user approves on a separate device/app, service polls for token.

## Rationale
- No browser needed on the requesting device
- User approves on their own device (phone, laptop)
- Polling model is simple to implement and debug
- 5-min expiry prevents stale requests

## Consequences
- Requires a separate approval mechanism (web page, push notification)
- Polling adds latency vs. push-based notification

---

# ADR-013: Token Binding via Client Fingerprint

**Status:** Accepted  
**Date:** 2026-04-12

## Context
Bearer tokens are susceptible to theft — if stolen, they can be used from any client.

## Decision
Bind tokens to client fingerprint (SHA-256 of IP + User-Agent) on first use. Subsequent requests must match.

## Rationale
- Prevents stolen token reuse from different network/client
- First-use binding requires no client-side changes
- Lightweight — no cryptographic proof required (DPoP is separate)

## Consequences
- Legitimate IP changes (VPN, mobile roaming) will invalidate the binding
- User-Agent changes (browser updates) will also invalidate

---

# ADR-014: W3C Traceparent Propagation

**Status:** Accepted  
**Date:** 2026-04-12

## Context
Distributed tracing across the proxy and upstream OpenSearch requires correlating spans. Without standard trace context propagation, each hop starts a new trace.

## Decision
Implement W3C Trace Context (traceparent header). Extract incoming trace-id, create child spans with same trace-id, inject traceparent in responses and upstream requests.

## Rationale
- W3C Trace Context is the industry standard (supported by Jaeger, Zipkin, OTEL)
- Enables end-to-end latency analysis across proxy → OpenSearch
- Zero config for clients already sending traceparent

## Consequences
- Adds ~1μs overhead per request for header parsing
- Clients not sending traceparent get proxy-generated trace IDs

---

# ADR-015: Multi-Tier Rate Limiting

**Status:** Accepted  
**Date:** 2026-04-12

## Context
Different clients and API keys need different rate limits. A single global limit is too coarse — admin endpoints need lower limits than read-only log queries.

## Decision
Three-tier rate limiting: per-client (by OAuth client_id), per-API-key (by key ID), and scope-based (admin scopes get lower RPM). Deny-overrides: most restrictive limit wins.

## Rationale
- Per-client prevents one client from starving others
- Per-API-key enables M2M rate control without OAuth overhead
- Scope-based limits protect admin endpoints from abuse
- Sliding window algorithm avoids burst-at-boundary problems

## Consequences
- Memory grows linearly with unique client/key count (mitigated by cleanup)
- Clients must handle 429 responses and Retry-After header

---

# ADR-016: JWT Access Tokens (RFC 9068)

**Status:** Accepted  
**Date:** 2026-04-12

## Context
Opaque tokens require a round-trip to the proxy for every validation. Resource servers in a distributed deployment need stateless token validation.

## Decision
Add optional JWT access tokens signed with the keyring RSA keys. Enabled via `jwt_access_token: true`. Token type is `at+jwt` per RFC 9068. Claims include iss, sub, client_id, scope, iat, exp, jti. kid header references the keyring key for JWKS-based validation.

## Rationale
- Stateless validation via JWKS endpoint
- Standard format understood by any JWT library
- Backward compatible — disabled by default
- Key rotation handled by existing keyring

## Consequences
- JWTs are larger than opaque tokens (~800 bytes vs ~40 bytes)
- Revocation requires introspection check (JWT is valid until exp)
- Token content is visible to anyone with the token (base64, not encrypted)

---

# ADR-017: Refresh Token Expiry and Absolute Lifetime

**Status:** Accepted  
**Date:** 2026-04-12

## Context
Refresh tokens had no expiry — a stolen refresh token could be used indefinitely. This violates security compliance requirements (SOC 2, PCI DSS).

## Decision
Two expiry mechanisms:
1. Per-token TTL (default 30 days) — each refresh token expires independently
2. Absolute family lifetime (default 90 days) — the entire rotation chain expires, forcing re-authentication

## Rationale
- TTL limits exposure window for stolen tokens
- Absolute lifetime prevents indefinite session extension via rotation
- Family revocation on absolute expiry forces clean re-auth
- Configurable via `refresh_token_ttl` and `refresh_max_life`

## Consequences
- Long-running services must handle re-authentication after 90 days
- Clients should monitor token expiry and re-auth proactively

---

# ADR-018: OIDC UserInfo Endpoint

**Status:** Accepted  
**Date:** 2026-04-12

## Decision
Implement GET/POST /oauth/userinfo per OIDC Core §5.3. Returns sub (client_id) and scope for valid Bearer tokens. 401 with WWW-Authenticate for invalid tokens.

## Rationale
- Required for OIDC compliance
- Enables standard OIDC client libraries to work with oauth4os

---

# ADR-019: DPoP Token Binding (RFC 9449)

**Status:** Accepted  
**Date:** 2026-04-12

## Decision
Bind access tokens to DPoP key thumbprints. BindDPoP stores the thumbprint, VerifyDPoP checks it with constant-time compare. JWT access tokens include cnf.jkt claim when bound. Unbound tokens pass verification (backward compatible).

## Rationale
- Prevents stolen token reuse without the DPoP private key
- Stronger than IP-based token binding (works across networks)
- Standard mechanism per RFC 9449

---

# ADR-020: Webhook HMAC-SHA256 Signatures

**Status:** Accepted  
**Date:** 2026-04-12

## Decision
Sign outgoing webhook event payloads with HMAC-SHA256. Signature sent in X-Webhook-Signature: sha256=<hex>. Receivers verify by computing HMAC with shared secret.

## Rationale
- Prevents webhook spoofing
- Industry standard (GitHub, Stripe, Slack all use HMAC-SHA256)
- Optional — no signature without key configured
