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
