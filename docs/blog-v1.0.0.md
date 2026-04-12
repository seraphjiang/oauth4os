# oauth4os v1.0.0 — Production Ready

**April 12, 2026** — oauth4os reaches v1.0.0. A complete OAuth 2.0 proxy for OpenSearch, built from zero in a single session.

## By the Numbers

| Metric | Value |
|--------|-------|
| Commits | 542 |
| Files | 443 |
| Test functions | 726 |
| Mutations killed | 117 |
| Packages | 50 |
| External dependencies | 2 |
| Lines of Go | ~22,000 |

## What oauth4os Does

oauth4os sits between your clients and OpenSearch. It validates OAuth 2.0 tokens, maps scopes to index-level permissions, and forwards requests — all in a single binary with zero external dependencies beyond `golang-jwt` and `gopkg.in/yaml.v3`.

```
Client → oauth4os → OpenSearch
           ↕
        OIDC Provider
```

## Key Features in v1.0.0

### Authentication
- **6 OAuth flows**: Authorization Code + PKCE, Client Credentials, Device Flow, Token Exchange, Introspection, API Keys
- **DPoP** (RFC 9449): Proof-of-possession tokens
- **Multi-provider**: Connect multiple OIDC providers simultaneously

### Authorization
- **Cedar policy engine**: Fine-grained access control with permit/forbid rules, glob patterns, conditions
- **Scope mapping**: Map OAuth scopes to OpenSearch index patterns (`read:logs-*` → read access on `logs-*`)
- **Multi-tenant**: Isolated policy sets per tenant with global policy inheritance

### Resilience
- **Circuit breaker**: Automatic upstream failure detection with half-open recovery
- **Rate limiting**: Per-scope configurable limits with Retry-After headers
- **Load shedding**: CPU-based admission control prevents cascade failures
- **Request deduplication**: Idempotency keys prevent duplicate writes
- **Graceful shutdown**: 30-second connection drain with `/ready` probe

### Observability
- **Prometheus metrics**: 16 metrics covering requests, cache, circuit breaker, upstream latency
- **OTLP tracing**: W3C Traceparent propagation with batched span export
- **Structured audit log**: Every auth decision logged with client, scope, policy, and outcome
- **Health endpoints**: `/health` (liveness), `/health/deep` (upstream + JWKS), `/ready` (readiness)

### Operations
- **26-command CLI**: `oauth4os-demo` for search, tail, watch, diff, stats, alerts, audit
- **Admin API**: Client CRUD, Cedar policy management, rate limit tuning, config backup/restore
- **Developer portal**: Interactive playground, analytics dashboard, OpenAPI docs
- **Docker Hub**: `docker run -p 8443:8443 jianghuan/oauth4os:latest`

### Security
- **SigV4 signing**: Native AWS OpenSearch Serverless (AOSS) support with credential caching
- **Consent screen**: Scope descriptions, write-scope warnings, 10-minute expiry, i18n (8 languages)
- **Security headers**: HSTS, X-Content-Type-Options, X-Frame-Options, Referrer-Policy

## Architecture Highlights

- **2 external deps** — everything else is Go stdlib
- **Embedded assets** — landing page, install script, demo app, OpenAPI spec all compiled into the binary
- **Stateless** — scale horizontally without coordination; tokens are self-contained JWTs
- **232 req/s** on 0.25 vCPU App Runner — Cedar evaluation at 114ns/op

## Try It

```bash
# Docker Hub
docker run -p 8443:8443 jianghuan/oauth4os:latest

# Live demo
open https://f5cmk2hxwx.us-west-2.awsapprunner.com

# CLI
curl -fsSL https://f5cmk2hxwx.us-west-2.awsapprunner.com/install.sh | sh
```

## What's Next

v1.0.0 is the foundation. The roadmap includes:
- WebAuthn/passkey authentication
- Policy playground (test Cedar policies before deploying)
- Grafana dashboard templates
- Helm chart for Kubernetes
- OpenSearch Dashboards plugin

## Links

- **GitHub**: https://github.com/seraphjiang/oauth4os
- **Docker Hub**: https://hub.docker.com/r/jianghuan/oauth4os
- **Live Demo**: https://f5cmk2hxwx.us-west-2.awsapprunner.com
- **RFC Discussion**: https://github.com/opensearch-project/.github/issues/491
