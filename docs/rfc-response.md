# RFC Comment — OAuth 2.0 Proxy for OpenSearch

> Ready to copy-paste into [opensearch-project/.github#491](https://github.com/opensearch-project/.github/issues/491)

---

Hi all — we built a working implementation of the OAuth 2.0 proxy approach described in this RFC. It's deployed and running against real OpenSearch (AOSS).

**🔗 Live demo**: https://f5cmk2hxwx.us-west-2.awsapprunner.com
**📦 Source**: https://github.com/seraphjiang/oauth4os
**📄 License**: Apache 2.0

Try it right now — no setup required:

```bash
# Get a scoped token
TOKEN=$(curl -sf -X POST https://f5cmk2hxwx.us-west-2.awsapprunner.com/oauth/token \
  -d "grant_type=client_credentials&client_id=demo&client_secret=secret&scope=read:logs-*" \
  | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

# Search logs through the proxy
curl -sf -H "Authorization: Bearer $TOKEN" \
  "https://f5cmk2hxwx.us-west-2.awsapprunner.com/logs-demo/_search?q=level:ERROR" | python3 -m json.tool
```

Or visit the [interactive demo app](https://f5cmk2hxwx.us-west-2.awsapprunner.com/demo) — it walks through the full PKCE browser flow with a consent screen.

---

## What oauth4os does

oauth4os is a reverse proxy that adds OAuth 2.0 token management to OpenSearch with **zero changes to OpenSearch itself**.

```
Clients → oauth4os (:8443) → OpenSearch (:9200)
```

It validates JWTs, maps OAuth scopes to OpenSearch roles, enforces Cedar authorization policies, and forwards authenticated requests. Existing OpenSearch auth continues to work — the proxy is additive.

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        oauth4os proxy                        │
│                                                              │
│  Tracing → Rate Limit → JWT Validate → Scope Map → Cedar    │
│     │          │             │             │          │      │
│     │          │         ┌───┘             │          │      │
│     │          │         ▼                 │          │      │
│     │          │    JWKS Cache ←── OIDC Providers     │      │
│     │          │    (auto-refresh)  (Keycloak/Dex/    │      │
│     │          │                    Auth0/Okta)       │      │
│     ▼          ▼                                      ▼      │
│  Audit Log  429+Retry-After              Permit/Forbid       │
│                                                              │
│  ──────────────── Reverse Proxy ─────────────────────────    │
│     │                    │                    │               │
│     ▼                    ▼                    ▼               │
│  OpenSearch Engine   Dashboards      Multi-cluster routing   │
└──────────────────────────────────────────────────────────────┘
```

## Implemented features

**Authentication (4 OAuth RFCs)**:
- JWT validation (RS256/ES256) with JWKS auto-discovery and caching
- PKCE authorization code flow for browser clients (RFC 7636) with consent screen
- Token introspection (RFC 7662)
- Token exchange — swap external IdP tokens for scoped proxy tokens (RFC 8693)
- Dynamic client registration with secret rotation (RFC 7591)
- Refresh token rotation with reuse detection
- OIDC Discovery endpoint (`/.well-known/openid-configuration`)
- RSA key rotation with JWKS endpoint (`/.well-known/jwks.json`)

**Authorization**:
- Scope-to-role mapping (per-tenant with global fallback)
- Cedar policy engine — permit/forbid rules, deny-overrides, multi-tenant
- Per-client rate limiting (token bucket, scope-aware RPM)
- Per-client IP allowlist/denylist (CIDR)
- Mutual TLS client authentication

**Operations**:
- Prometheus metrics endpoint (`/metrics`)
- OpenTelemetry-style distributed tracing (X-Trace-ID, span per stage)
- Structured JSON audit logging with query support
- Token analytics dashboard (top clients, scope distribution, error rates)
- Session management (list, revoke, force logout)
- Admin REST API for live config changes (scopes, policies, rate limits)
- Config backup/restore bundles

**Enterprise**:
- Multi-cluster federation — route to N OpenSearch clusters by index pattern
- AWS SigV4 signing for OpenSearch Serverless (AOSS)
- Multi-tenant by OIDC issuer — each provider gets its own scope mappings and policies

**Developer experience**:
- CLI tool (`oauth4os login`, `create-token`, `search`, `services`)
- MCP server reference (7 tools for AI agents — Claude, LangChain, etc.)
- OpenSearch Dashboards plugin for token management
- Interactive demo app with PKCE flow
- One-line install: `curl -sL <proxy>/install.sh | bash`

**Deployment**:
- Docker + docker-compose
- Helm chart
- AWS CDK stack
- AWS App Runner (current live demo)
- GitHub Actions CI
- Single binary, zero external dependencies (stdlib + 2 libraries)

## By the numbers

| Metric | Value |
|--------|-------|
| Go source (non-test) | 9,000 lines |
| Test code | 9,500 lines |
| Test functions | 558 |
| Internal packages | 46 |
| Commits | 442 |
| OAuth RFCs implemented | 4 (7636, 7662, 8693, 7591) |
| External dependencies | 2 (jwt, yaml) |

## Key design decisions

1. **Zero changes to OpenSearch** — The proxy translates OAuth scopes to OpenSearch Security Plugin backend roles via `X-Proxy-User` and `X-Proxy-Roles` headers. No patches, no forks. Existing auth methods continue to work alongside the proxy.

2. **Cedar for fine-grained authz** — Cedar policies support deny-overrides (e.g., "permit all reads, but forbid `.opendistro_security` index access") and compose cleanly per-tenant. This is more expressive than flat RBAC while remaining auditable.

3. **Multi-tenant by OIDC issuer** — Each identity provider gets its own scope mappings and Cedar policies. A Keycloak realm and a Dex instance can coexist with completely different authorization rules.

4. **Token exchange as the federation story** — RFC 8693 lets external IdP users swap their JWT for a scoped oauth4os token. This is the key enabler for machine-to-machine access without sharing OpenSearch credentials.

5. **MCP server as the AI story** — The reference MCP server demonstrates how AI agents can securely query OpenSearch with scoped, auditable, revocable tokens — a pattern that's increasingly important as LLM-based tools access production data.

6. **Stdlib-only philosophy** — Only 2 external dependencies (jwt parsing, YAML config). Everything else — HTTP routing, TLS, crypto, rate limiting, tracing — uses Go's standard library. This minimizes supply chain risk and keeps the binary small.

## How to run it locally

```bash
git clone https://github.com/seraphjiang/oauth4os
cd oauth4os
docker compose up
```

Or try the live demo:
- **Landing page**: https://f5cmk2hxwx.us-west-2.awsapprunner.com
- **OIDC Discovery**: https://f5cmk2hxwx.us-west-2.awsapprunner.com/.well-known/openid-configuration
- **Health**: https://f5cmk2hxwx.us-west-2.awsapprunner.com/health
- **Demo app** (PKCE flow): https://f5cmk2hxwx.us-west-2.awsapprunner.com/demo

## Questions for the community

1. **Deployment model** — Should this be a standalone proxy (current), an OpenSearch plugin, or both? The proxy approach has the advantage of zero OpenSearch changes, but a plugin could offer tighter integration.

2. **Scope format** — We used `read:<index-pattern>` / `write:<index-pattern>` / `admin`. Does this align with how teams model OpenSearch access? Should we support OpenSearch's existing action groups?

3. **Cedar vs. simpler RBAC** — Is Cedar's expressiveness worth the learning curve, or would a simpler role-mapping suffice for most deployments?

4. **Upstream path** — If there's interest, we'd like to contribute this upstream. What would the right integration point be — Security Plugin extension, standalone project under opensearch-project, or something else?

We'd love feedback. Happy to demo, walk through the architecture, or pair on integration.

---

*Built as a proof-of-concept for this RFC. All code is Apache 2.0 licensed.*
