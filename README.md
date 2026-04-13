# oauth4os — OAuth 2.0 Proxy for OpenSearch

**Secure machine-to-machine access for OpenSearch.** OAuth 2.0 proxy that validates JWT tokens, maps scopes to OpenSearch security roles, and forwards requests — with zero changes to OpenSearch itself.

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go 1.22](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://go.dev)
[![CI](https://github.com/seraphjiang/oauth4os/actions/workflows/ci.yml/badge.svg)](https://github.com/seraphjiang/oauth4os/actions/workflows/ci.yml)
[![Release](https://github.com/seraphjiang/oauth4os/actions/workflows/release.yml/badge.svg)](https://github.com/seraphjiang/oauth4os/releases)
[![Docker](https://img.shields.io/badge/Docker-jianghuan%2Foauth4os-2496ED?logo=docker)](https://hub.docker.com/r/jianghuan/oauth4os)
[![OpenSearch 2.x | 3.x](https://img.shields.io/badge/OpenSearch-2.x%20%7C%203.x-orange?logo=opensearch)](https://opensearch.org)

> 🔗 **RFC**: [opensearch-project/.github#491](https://github.com/opensearch-project/.github/issues/491)
> · 🎯 **Live Demo**: [f5cmk2hxwx.us-west-2.awsapprunner.com](https://f5cmk2hxwx.us-west-2.awsapprunner.com)
> · 🔍 **Demo App**: [/demo](https://f5cmk2hxwx.us-west-2.awsapprunner.com/demo) (interactive PKCE flow)
> · 📖 **Docs**: [docs/](docs/)

**Try it now — no setup required:**
```bash
# Get a scoped token
TOKEN=$(curl -sf -X POST https://f5cmk2hxwx.us-west-2.awsapprunner.com/oauth/token \
  -d "grant_type=client_credentials&client_id=demo-agent&client_secret=demo-secret&scope=read:logs-*" \
  | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

# Search real logs through the proxy (backed by OpenSearch Serverless)
curl -sf -H "Authorization: Bearer $TOKEN" \
  "https://f5cmk2hxwx.us-west-2.awsapprunner.com/logs-demo/_search?q=level:ERROR" | python3 -m json.tool

# Or install the CLI
curl -sL https://f5cmk2hxwx.us-west-2.awsapprunner.com/install.sh | bash
oauth4os-demo login
oauth4os-demo search 'level:ERROR'
```

---

## Project Stats

| Metric | Value |
|--------|-------|
| Commits | 889 |
| Files | 609 |
| Go source (non-test) | 9,000 lines |
| Test code | 9,500 lines |
| Test functions | 1196 |
| Test packages passing | 65/65 |
| Internal packages | 65 |
| OAuth RFCs implemented | 4 (7636, 7662, 8693, 7591) |
| External dependencies | 2 (jwt, yaml) |
| Throughput | 232 req/s (0.25 vCPU) |

---

## Why?

OpenSearch has OIDC auth and API Keys (3.7), but lacks the developer experience layer for machine-to-machine access:

| Capability | Grafana | Datadog | Elastic | OpenSearch | + oauth4os |
|---|:---:|:---:|:---:|:---:|:---:|
| OIDC / SSO | ✅ | ✅ | ✅ | ✅ | ✅ |
| API Keys | ✅ | ✅ | ✅ | 🔄 3.7 | ✅ |
| OAuth Apps / Scoped Tokens | ✅ | ✅ | ✅ | ❌ | **✅** |
| Token Governance UI | ✅ | ✅ | ✅ | ❌ | **✅** |
| Rate Limiting (per-client) | ✅ | ✅ | ✅ | ❌ | **✅** |
| Cedar Fine-Grained Policies | ❌ | ❌ | ❌ | ❌ | **✅** |
| Token Introspection (RFC 7662) | ❌ | ❌ | ✅ | ❌ | **✅** |
| PKCE for Browser Clients | ✅ | ❌ | ✅ | ❌ | **✅** |
| Multi-Tenancy | ✅ | ✅ | ✅ | ⚠️ | **✅** |

## Quick Start

```bash
# Option 1: Docker (recommended)
docker run -p 8443:8443 jianghuan/oauth4os:latest
# Open http://localhost:8443

# Option 2: Clone + Docker Compose
git clone https://github.com/seraphjiang/oauth4os
cd oauth4os && docker compose up

# Option 3: CLI
curl -sL https://f5cmk2hxwx.us-west-2.awsapprunner.com/install.sh | bash
oauth4os-demo login
oauth4os-demo search 'level:ERROR'
```

Then get a token and query:

```bash
# Get a scoped token
TOKEN=$(curl -sf -X POST http://localhost:8443/oauth/token \
  -d "grant_type=client_credentials&client_id=demo-agent&client_secret=demo-secret&scope=read:logs-*" \
  | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

# Query OpenSearch through the proxy
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/logs-*/_search \
  -d '{"query": {"match": {"level": "error"}}}'

# Revoke when done
curl -X DELETE http://localhost:8443/oauth/token/<token-id>
```

## Architecture

```
┌─────────────┐     ┌──────────────────────────┐     ┌─────────────────┐
│  Clients    │     │      oauth4os proxy       │     │   OpenSearch    │
│             │     │      (Go, :8443)          │     │                 │
│  AI Agent   │────▶│  Tracing                  │────▶│  Engine :9200   │
│  CI/CD      │     │  Rate Limiting            │     │                 │
│  Slack Bot  │     │  JWT Validation           │────▶│  Dashboards     │
│  CLI        │     │  Scope → Role Mapping     │     │  :5601          │
│  MCP Server │     │  Cedar Policies           │     │                 │
│  Browser    │     │  IP Filter / mTLS         │────▶│  AOSS (SigV4)  │
│  Demo App   │     │  Audit Logging            │     │                 │
└─────────────┘     │  Multi-cluster Routing    │     └─────────────────┘
                    └────────────┬───────────────┘
                                │
                    ┌───────────▼────────────┐
                    │    OIDC Providers      │
                    │  Keycloak · Auth0      │
                    │  Okta · Dex · Google   │
                    └────────────────────────┘
```

## Features

### Authentication (4 OAuth RFCs)
- **JWT validation** — JWKS auto-discovery, RS256/ES256, configurable clock skew
- **PKCE authorization code flow** with consent screen (RFC 7636)
- **Token introspection** (RFC 7662)
- **Token exchange** — swap external IdP tokens for scoped proxy tokens (RFC 8693)
- **Dynamic client registration** with secret rotation (RFC 7591)
- **Refresh token rotation** with reuse detection and sliding window expiry
- **OIDC Discovery** (`/.well-known/openid-configuration`)
- **RSA key rotation** with JWKS endpoint (`/.well-known/jwks.json`)

### Authorization
- **Scope-to-role mapping** — `read:logs-*` → OpenSearch `logs_reader` role
- **Cedar policy engine** — permit/forbid rules, deny-overrides, multi-tenant
- **Per-client rate limiting** — token bucket, scope-aware RPM, `429 + Retry-After`
- **Per-client IP allowlist/denylist** — CIDR-based filtering
- **Mutual TLS** — client certificate authentication

### Operations
- **Prometheus metrics** — `/metrics` (requests, auth, Cedar, upstream errors)
- **Distributed tracing** — OpenTelemetry-style, X-Trace-ID, span per stage
- **Structured JSON audit logging** with query support
- **Token analytics dashboard** — top clients, scope distribution, error rates
- **Session management** — list active sessions, revoke, force logout
- **Admin REST API** — live config changes (scopes, policies, rate limits)
- **Config backup/restore** bundles

### Enterprise
- **Multi-cluster federation** — route to N OpenSearch clusters by index pattern
- **AWS SigV4 signing** for OpenSearch Serverless (AOSS)
- **Multi-tenant** by OIDC issuer — each provider gets its own scope mappings and policies
- **Client CRUD** — create, list, update, delete clients with secret rotation
- **API key authentication** — X-API-Key header support with per-key rate limits
- **Circuit breaker** — automatic upstream failure detection and recovery
- **Response caching** — TTL-based GET cache, configurable per-route
- **Load shedding** — reject requests when queue depth exceeds threshold
- **Request retry** — exponential backoff for upstream 5xx (max 3 retries)

### Developer Experience
- **Developer portal** — `/developer/docs` (OpenAPI), `/developer/analytics`
- **Demo web app** — `/demo` — log viewer with PKCE login, search, scope enforcement demo
- **CLI installer** — `curl -sL <proxy>/install.sh | bash` → `oauth4os-demo login/search/services`
- **CLI tool** — `oauth4os login`, `create-token`, `revoke`, `status`
- **MCP server** — reference integration for AI agents (7 tools)
- **OSD plugin** — token management UI in OpenSearch Dashboards
- **Consent screen** — localized in 8 languages (en, es, fr, de, ja, zh, pt, ko)
- **Token inspector UI** — `/admin/tokens` visual token details
- **Config editor UI** — `/admin/config` edit config from browser

### Resilience
- **CORS middleware** — configurable origin allowlist
- **Idempotency** — request deduplication via Idempotency-Key header
- **DPoP token binding** — bind tokens to client fingerprint (RFC 9449 prep)
- **Device authorization flow** — for CLI/IoT devices without browser (RFC 8628)
- **OTLP export** — send spans to OpenTelemetry Collector
- **Structured access logs** — JSON per request with method, path, status, latency

### Deployment
- **Docker** + docker-compose
- **Helm chart**
- **AWS CDK stack**
- **AWS App Runner** with auto-deploy on ECR push
- **GitHub Actions CI** — build, test, vet, Docker build on push; release on tag
- **Single binary** — zero external dependencies (stdlib + 2 libraries)

## Demo Screenshots

### Landing Page
Visit [f5cmk2hxwx.us-west-2.awsapprunner.com](https://f5cmk2hxwx.us-west-2.awsapprunner.com) — dark/light theme, feature comparison, architecture diagram, interactive try-it-now section.

### Consent Screen
`GET /oauth/authorize` → shows app name, requested scopes with descriptions, approve/deny buttons. Write scopes trigger a warning banner.

### Demo Log Viewer
`GET /demo` → login with PKCE → search logs by service/level, scope enforcement demo (read ✅ vs write ❌).

### Analytics Dashboard
`GET /developer/analytics` — top clients, scope distribution, request timeline, error rates. Auto-refreshes every 5s.

## Live Demo URLs

| URL | Description |
|-----|-------------|
| [/](https://f5cmk2hxwx.us-west-2.awsapprunner.com) | Landing page |
| [/demo](https://f5cmk2hxwx.us-west-2.awsapprunner.com/demo) | Log viewer demo app (PKCE flow) |
| [/health](https://f5cmk2hxwx.us-west-2.awsapprunner.com/health) | Health check + version |
| [/metrics](https://f5cmk2hxwx.us-west-2.awsapprunner.com/metrics) | Prometheus metrics |
| [/.well-known/openid-configuration](https://f5cmk2hxwx.us-west-2.awsapprunner.com/.well-known/openid-configuration) | OIDC Discovery |
| [/.well-known/jwks.json](https://f5cmk2hxwx.us-west-2.awsapprunner.com/.well-known/jwks.json) | JWKS |
| [/developer/docs](https://f5cmk2hxwx.us-west-2.awsapprunner.com/developer/docs) | OpenAPI documentation |
| [/playground](https://f5cmk2hxwx.us-west-2.awsapprunner.com/playground) | Interactive OAuth playground |
| [/analytics](https://f5cmk2hxwx.us-west-2.awsapprunner.com/analytics) | Token analytics dashboard |
| [/version](https://f5cmk2hxwx.us-west-2.awsapprunner.com/version) | Version info (JSON) |
| [/install.sh](https://f5cmk2hxwx.us-west-2.awsapprunner.com/install.sh) | CLI installer script |

## Project Structure

```
cmd/
  proxy/              — Main proxy binary (with embedded landing page)
  cli/                — CLI tool (login, create-token, revoke, status)
internal/
  accesslog/          — Structured JSON access logs per request
  admin/              — Admin REST API (config CRUD, backup/restore)
  analytics/          — Token usage analytics tracker
  apikey/             — API key authentication
  audit/              — Structured JSON audit logging
  backup/             — Config backup/restore bundles
  cache/              — TTL-based response cache for GET requests
  cedar/              — Cedar policy engine (multi-tenant)
  circuit/            — Circuit breaker for upstream failures
  config/             — YAML config loader
  configui/           — Browser-based config editor (/admin/config)
  cors/               — CORS middleware
  demo/               — Demo web app backend (PKCE callback)
  device/             — OAuth 2.0 Device Authorization (RFC 8628)
  discovery/          — OIDC Discovery endpoint
  dpop/               — DPoP token binding (RFC 9449 prep)
  events/             — Webhook notifications for token events
  exchange/           — RFC 8693 token exchange
  federation/         — Multi-cluster routing
  healthcheck/        — Background upstream health checker
  i18n/               — Consent screen localization (8 languages)
  idempotency/        — Request deduplication via Idempotency-Key
  introspect/         — RFC 7662 token introspection
  ipfilter/           — Per-client IP allowlist/denylist
  jwt/                — JWT validation + JWKS cache
  keyring/            — RSA key rotation + JWKS
  loadshed/           — Load shedding (reject when queue full)
  logging/            — Structured logging
  mtls/               — Mutual TLS client auth
  otlp/               — OpenTelemetry OTLP span exporter
  par/                — Pushed Authorization Requests
  pkce/               — PKCE authorization + consent screen
  ratelimit/          — Per-client token bucket
  registration/       — Dynamic client registration (RFC 7591)
  retry/              — Exponential backoff for upstream 5xx
  scope/              — Scope-to-role mapping engine
  session/            — Session management
  sigv4/              — AWS SigV4 signing for AOSS
  token/              — Token lifecycle (issue/refresh/revoke)
  tokenbind/          — Token-to-client fingerprint binding
  tokenui/            — Token inspector UI
  tracing/            — OpenTelemetry-style distributed tracing
  webhook/            — Webhook authorizer
```

## Configuration

```yaml
upstream:
  engine: https://opensearch:9200
  dashboards: https://opensearch-dashboards:5601

providers:
  - name: keycloak
    issuer: https://keycloak.example.com/realms/opensearch
    jwks_uri: auto

scope_mapping:
  "read:logs-*":
    backend_roles: [logs_read_access]
  "write:dashboards":
    backend_roles: [dashboard_write_access]
  "admin":
    backend_roles: [all_access]

rate_limits:
  default_rpm: 60
  per_scope:
    "read:logs-*": 120
    "admin": 30

ip_filter:
  clients:
    my-agent:
      allow: ["10.0.0.0/8"]

mtls:
  enabled: false
  ca_file: /etc/oauth4os/ca.pem

listen: :8443
```

## API Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/oauth/token` | POST | Issue token (client_credentials, refresh_token, authorization_code) |
| `/oauth/tokens` | GET | List active tokens |
| `/oauth/token/{id}` | GET/DELETE | Get or revoke token |
| `/oauth/introspect` | POST | Token introspection (RFC 7662) |
| `/oauth/authorize` | GET | PKCE authorization (consent screen) |
| `/oauth/consent` | POST | Approve/deny consent |
| `/oauth/register` | POST/GET | Dynamic client registration (RFC 7591) |
| `/oauth/register/{id}` | GET/PUT/DELETE | Client CRUD |
| `/oauth/register/{id}/rotate` | POST | Rotate client secret |
| `/admin/analytics` | GET | Token usage analytics |
| `/admin/audit` | GET | Audit log query |
| `/admin/sessions` | GET | List active sessions |
| `/admin/clusters` | GET | Multi-cluster status |
| `/health` | GET | Health check + version |
| `/health/deep` | GET | Deep health (upstream, JWKS, TLS) |
| `/metrics` | GET | Prometheus metrics |
| `/.well-known/openid-configuration` | GET | OIDC Discovery |
| `/.well-known/jwks.json` | GET | JWKS endpoint |
| `/*` | ANY | Reverse proxy to OpenSearch (with auth) |

## Documentation

| Doc | Description |
|-----|-------------|
| [docs/architecture.md](docs/architecture.md) | Architecture, data flow, component diagrams |
| [docs/security.md](docs/security.md) | Threat model, auth flows, JWT validation |
| [docs/user-manual.md](docs/user-manual.md) | Complete user manual |
| [docs/rfc-response.md](docs/rfc-response.md) | RFC comment for opensearch-project/.github#491 |
| [docs/blog-post.md](docs/blog-post.md) | "Building an OAuth 2.0 Proxy for OpenSearch" |
| [docs/adr/](docs/adr/) | 10 Architecture Decision Records |
| [CHANGELOG.md](CHANGELOG.md) | Version history |
| [docs/api-reference.md](docs/api-reference.md) | All 35 endpoints with curl examples |
| [docs/sdk-guide.md](docs/sdk-guide.md) | Go, Python, Node, Rust, Java integration |
| [docs/cedar-guide.md](docs/cedar-guide.md) | Cedar policy syntax, examples, multi-tenant |
| [docs/deployment.md](docs/deployment.md) | App Runner, ECS, Kubernetes, Docker |
| [docs/performance.md](docs/performance.md) | Benchmarks, tuning, capacity planning |
| [docs/migration.md](docs/migration.md) | Migrate from basic auth, API keys, nginx |
| [docs/troubleshooting.md](docs/troubleshooting.md) | Common issues and fixes |
| [docs/faq.md](docs/faq.md) | 20 most common questions |
| [docs/monitoring.md](docs/monitoring.md) | Prometheus + Grafana setup |
| [docs/canary-deployment.md](docs/canary-deployment.md) | Canary deployment guide |
| [docs/opentelemetry.md](docs/opentelemetry.md) | OTLP tracing integration |
| [docs/runbook.md](docs/runbook.md) | Operator runbook — incidents, scaling |
| [docs/quickstart.md](docs/quickstart.md) | 5-minute setup guide |
| [docs/getting-started.md](docs/getting-started.md) | Hands-on tutorial — 7 steps |
| [docs/blog-v2.0.0.md](docs/blog-v2.0.0.md) | v2.0.0 release blog — 32 features |
| [docs/testing.md](docs/testing.md) | Test strategy and coverage |
| [docs/benchmarks.md](docs/benchmarks.md) | Performance benchmarks |
| [docs/comparison.md](docs/comparison.md) | oauth4os vs alternatives |
| [docs/oauth-flows.md](docs/oauth-flows.md) | All 8 OAuth flows with diagrams |

## Contributing

Pull requests welcome. Please open an issue first to discuss major changes.

1. Fork the repo
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Run tests: `go test ./...`
4. Commit following [Conventional Commits](https://www.conventionalcommits.org/)
5. Open a pull request

## License

Apache 2.0 — see [LICENSE](LICENSE).
