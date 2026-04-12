# Building an OAuth 2.0 Proxy for OpenSearch in a Weekend

*How we built a production-grade OAuth proxy with Cedar policies, token exchange, and AI agent integration — in Go, from scratch.*

---

OpenSearch has solid authentication. OIDC, SAML, API keys (coming in 3.7). But if you've tried to give an AI agent scoped, revocable access to your cluster — or federate tokens across multiple identity providers — you've hit the wall.

We built [oauth4os](https://github.com/seraphjiang/oauth4os) to fill that gap. It's a reverse proxy that adds OAuth 2.0 token management to OpenSearch with zero changes to the cluster itself.

This is the story of how we built it.

## The Problem

We were building [S3 O11y](https://github.com/admin/s3-o11y), an AI-powered observability platform. Our AI chat agent needed to query OpenSearch — but we didn't want to hand it admin credentials. We wanted:

- **Scoped tokens**: The agent gets `read:logs-*`, not `all_access`
- **Revocable**: Kill a token instantly if compromised
- **Auditable**: Know exactly what the agent queried
- **Federated**: Accept tokens from any OIDC provider

OpenSearch's Security Plugin doesn't support this. Neither does any existing proxy we could find.

So we built one.

## Architecture: A Proxy, Not a Plugin

The key insight: **don't modify OpenSearch**. Put a proxy in front.

```
Client → oauth4os (:8443) → OpenSearch (:9200)
```

Every request passes through 6 stages:

1. **Tracing** — Assign X-Trace-ID, start span
2. **Rate limiting** — Token bucket per client, scope-aware
3. **JWT validation** — JWKS auto-discovery, RS256/ES256
4. **Scope mapping** — `read:logs-*` → OpenSearch `logs_read_access` role
5. **Cedar policy** — Fine-grained deny rules (e.g., block `.opendistro_security`)
6. **Audit** — Structured JSON log line

If any stage fails, the request never reaches OpenSearch. If all pass, the proxy forwards with `X-Proxy-User` and `X-Proxy-Roles` headers that OpenSearch Security Plugin trusts.

## Day 1: Core Proxy + JWT + Tokens

We started with the basics:

```go
// 50 lines to get a working reverse proxy
engineURL, _ := url.Parse("http://opensearch:9200")
proxy := httputil.NewSingleHostReverseProxy(engineURL)
```

Then JWT validation. The tricky part is JWKS caching — you don't want to fetch keys on every request, but you need to handle key rotation. We cache keys and refresh on signature failure.

Token management was straightforward: issue tokens via `POST /oauth/token` with `client_credentials` grant, store in memory, return Bearer tokens. Each token has scopes, expiry, and a refresh token.

By end of day 1: working proxy with JWT auth and token issuance.

## Day 1.5: Cedar Policies

RBAC wasn't enough. We needed "permit everything, except the security index." That's a deny-override pattern — exactly what Cedar is designed for.

Our Cedar engine is ~150 lines of Go. Two default policies:

```
permit(principal, action, resource);                    // allow all authenticated requests
forbid(principal, action, resource == ".opendistro_security");  // except the security index
```

Cedar policies are composable per-tenant. A Keycloak realm can have different rules than a Dex instance.

## Day 2: RFC Compliance + Production Hardening

We implemented four OAuth RFCs:

| RFC | What | Why |
|-----|------|-----|
| 7636 | PKCE | Browser clients need auth code flow without client secrets |
| 7662 | Token Introspection | Resource servers need to validate tokens |
| 8693 | Token Exchange | External IdP users swap their JWT for a scoped proxy token |
| 7591 | Dynamic Client Registration | Clients self-register via API |

Token exchange (RFC 8693) is the most interesting. An AI agent authenticates with its IdP, gets a JWT, then exchanges it for an oauth4os token scoped to `read:logs-*`. The agent never sees OpenSearch credentials.

```bash
# Exchange an external JWT for a scoped token
curl -X POST http://localhost:8443/oauth/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=<external-jwt>" \
  -d "scope=read:logs-*"
```

Production hardening: connection pooling (100 idle conns), graceful shutdown (30s drain), request timeouts, Prometheus metrics, structured JSON logging, OpenTelemetry tracing.

## The AI Agent Story: MCP Server

The killer demo is the [MCP server](https://github.com/seraphjiang/oauth4os/tree/main/examples/mcp-server). It's a Python server that implements the [Model Context Protocol](https://modelcontextprotocol.io/), giving AI agents like Claude secure access to OpenSearch.

7 tools: `search_logs`, `aggregate`, `get_indices`, `get_mappings`, `create_index`, `delete_docs`, `get_cluster_health`.

Add it to Claude Desktop:

```json
{
  "mcpServers": {
    "opensearch": {
      "command": "python3",
      "args": ["server.py"],
      "env": {
        "OAUTH4OS_URL": "http://localhost:8443",
        "OAUTH4OS_CLIENT_ID": "claude-agent",
        "OAUTH4OS_SCOPE": "read:logs-*"
      }
    }
  }
}
```

Now Claude can search your logs, run aggregations, and inspect mappings — all through scoped, auditable, revocable tokens. No admin credentials exposed.

## Multi-Tenancy

Each OIDC provider gets its own scope mappings and Cedar policies:

```yaml
tenants:
  "https://keycloak.acme.com/realms/ops":
    scope_mapping:
      "read:logs-*":
        backend_roles: [ops_logs_reader]
  "https://dex.partner.com":
    scope_mapping:
      "read:metrics-*":
        backend_roles: [partner_metrics_reader]
```

The Acme ops team sees logs. The partner sees metrics. Same proxy, same OpenSearch cluster, complete isolation.

## Admin API: No Restarts

Everything is configurable at runtime via REST:

```bash
# Add a Cedar policy
curl -X POST http://localhost:8443/admin/cedar-policies \
  -d '{"id": "block-pii", "effect": "forbid", "resource": "pii-*"}'

# Update rate limits
curl -X PUT http://localhost:8443/admin/rate-limits \
  -d '{"read:logs-*": 1000, "admin": 30}'

# Add a new OIDC provider
curl -X POST http://localhost:8443/admin/providers \
  -d '{"name": "okta", "issuer": "https://dev-123.okta.com"}'
```

No YAML editing. No proxy restarts. Changes take effect immediately.

## What We Learned

**Go's stdlib is enough.** `net/http`, `net/http/httputil`, `crypto` — we didn't need any web framework. The proxy is ~250 lines. The entire project is ~3,900 lines of non-test Go.

**Cedar is worth the complexity.** We considered simple RBAC but kept hitting edge cases. "Allow everything except X" is surprisingly common. Cedar handles it cleanly.

**Token exchange is the federation primitive.** Once you have RFC 8693, any IdP can participate. The proxy becomes a token translation layer.

**220 tests in a weekend is possible** when you write tests alongside features, not after. Every package has `_test.go` files. We have unit tests, integration tests, e2e tests, fuzz tests, chaos tests, and benchmarks.

## Try It

```bash
git clone https://github.com/seraphjiang/oauth4os
cd oauth4os
docker compose up
```

Get a token, query OpenSearch, revoke the token. Three curl commands.

We're proposing this approach to the OpenSearch community via [RFC #491](https://github.com/opensearch-project/.github/issues/491). If scoped tokens, token exchange, and AI agent access matter to you — we'd love your feedback.

---

*oauth4os is Apache 2.0 licensed. Contributions welcome.*
