# oauth4os vs OpenSearch Auth Options — Comparison Guide

This document compares oauth4os with the built-in authentication and authorization options available in OpenSearch.

## Overview

| | Security Plugin (OIDC) | Security Plugin (SAML) | API Keys (3.7+) | **oauth4os** |
|---|---|---|---|---|
| **Primary use case** | Human SSO | Human SSO | Machine access | Machine + human access |
| **Token type** | OIDC ID token | SAML assertion | Opaque key | OAuth 2.0 Bearer (JWT) |
| **Scoped access** | ❌ Full role | ❌ Full role | ⚠️ Role-based | ✅ Fine-grained scopes |
| **Token revocation** | ❌ Wait for expiry | ❌ Wait for expiry | ✅ Delete key | ✅ Instant revoke |
| **Token introspection** | ❌ | ❌ | ❌ | ✅ RFC 7662 |
| **Rate limiting** | ❌ | ❌ | ❌ | ✅ Per-client RPM |
| **Audit trail** | ⚠️ Audit log plugin | ⚠️ Audit log plugin | ⚠️ Audit log plugin | ✅ Built-in structured JSON |
| **Cedar policies** | ❌ | ❌ | ❌ | ✅ Fine-grained |
| **Multi-tenancy** | ⚠️ Tenant plugin | ⚠️ Tenant plugin | ❌ | ✅ Per-provider isolation |
| **Management UI** | ❌ | ❌ | ❌ | ✅ OSD plugin |
| **CLI tool** | ❌ | ❌ | ❌ | ✅ oauth4os CLI |
| **Breaking changes** | None | None | None | **None** |

## When to Use What

### Use the Security Plugin (OIDC) when:
- You only need human users to log in via SSO
- Your OIDC provider handles all access control
- You don't need per-token scoping or revocation
- You're already using the Security Plugin and it meets your needs

### Use the Security Plugin (SAML) when:
- Your organization mandates SAML (e.g., corporate IdP)
- You only need human SSO, not machine-to-machine access
- You don't need token lifecycle management

### Use API Keys (OpenSearch 3.7+) when:
- You need simple machine access without OAuth complexity
- You're on OpenSearch 3.7+ and can use the native feature
- You don't need scoped access, rate limiting, or Cedar policies
- You're comfortable managing keys manually

### Use oauth4os when:
- You need **machine-to-machine access** (AI agents, CI/CD, Slack bots)
- You want **scoped tokens** — `read:logs-*` instead of full admin
- You need **token governance** — list, inspect, revoke from a UI or CLI
- You want **rate limiting** per client to protect your cluster
- You need **Cedar policies** for fine-grained access control
- You want a **unified auth layer** across Engine + Dashboards
- You need **multi-tenancy** — different providers with different policies
- You want **audit logging** without configuring the audit log plugin

## Architecture Comparison

### Security Plugin (OIDC)

```
Browser → OpenSearch Dashboards → OIDC Provider → ID Token → Security Plugin → OpenSearch
```

- Token validated by Security Plugin directly
- Roles mapped from OIDC claims
- No proxy layer, no additional latency
- Limited to OIDC ID tokens

### oauth4os

```
Client → oauth4os proxy (:8443) → OpenSearch Engine/Dashboards
                ↓
         OIDC Provider (JWKS)
```

- Proxy validates JWT, maps scopes to roles, evaluates Cedar policies
- Adds ~1-2ms latency per request (JWT cache hit)
- Supports any client type (not just browsers)
- Token lifecycle management built in

## Feature Deep Dive

### Scoped Access

**Security Plugin**: Maps OIDC claims to backend roles. A user gets all permissions of their role — no per-request scoping.

**oauth4os**: Tokens carry specific scopes (`read:logs-*`, `write:dashboards`). The proxy maps scopes to the minimum required roles. A CI/CD pipeline gets only what it needs.

```yaml
# oauth4os scope mapping
scope_mapping:
  "read:logs-*":
    backend_roles: [logs_read_access]
  "write:dashboards":
    backend_roles: [dashboard_write_access]
```

### Token Revocation

**Security Plugin**: OIDC tokens expire based on TTL. No way to revoke a specific token before expiry.

**API Keys**: Can be deleted, but no instant propagation — cached keys may still work briefly.

**oauth4os**: Instant revocation via API or CLI. Revoked tokens are rejected on the next request.

```bash
# Instant revoke
oauth4os revoke tok_abc123
# or
curl -X DELETE http://proxy:8443/oauth/token/tok_abc123
```

### Rate Limiting

**Security Plugin / API Keys**: No built-in rate limiting. Must use external tools (nginx, AWS WAF).

**oauth4os**: Per-client token bucket rate limiter. Configurable RPM per scope. Returns `429 Retry-After`.

```yaml
rate_limit:
  default_rpm: 60
  per_scope:
    "read:logs-*": 120
    "admin": 30
```

### Cedar Policies

Unique to oauth4os. Fine-grained access control evaluated locally (no network calls):

```
permit(
  principal == "ci-pipeline",
  action == "GET",
  resource.index == "logs-*"
);

forbid(
  principal,
  action,
  resource.index == ".opendistro_security"
);
```

## Can They Coexist?

**Yes.** oauth4os is additive — it doesn't replace the Security Plugin. Existing OIDC/SAML/basic auth continues to work. oauth4os adds a proxy layer for machine-to-machine access alongside your existing human auth.

```
Humans  → OSD → Security Plugin (OIDC/SAML) → OpenSearch
Machines → oauth4os proxy → OpenSearch (same cluster)
```

## Migration Path

See [migration.md](migration.md) for a step-by-step guide to adding oauth4os alongside your existing Security Plugin setup.
