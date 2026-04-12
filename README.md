# oauth4os — OAuth 2.0 Proxy for OpenSearch

**Secure machine-to-machine access for OpenSearch.** OAuth 2.0 proxy that validates JWT tokens, maps scopes to OpenSearch security roles, and forwards requests to both Engine and Dashboards — with zero changes to existing components.

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go 1.22](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://go.dev)
[![CI](https://github.com/seraphjiang/oauth4os/actions/workflows/ci.yml/badge.svg)](https://github.com/seraphjiang/oauth4os/actions/workflows/ci.yml)
[![Release](https://github.com/seraphjiang/oauth4os/actions/workflows/release.yml/badge.svg)](https://github.com/seraphjiang/oauth4os/releases)
[![Docker](https://img.shields.io/badge/Docker-ghcr.io-2496ED?logo=docker)](https://github.com/seraphjiang/oauth4os/pkgs/container/oauth4os)
[![OpenSearch 2.x | 3.x](https://img.shields.io/badge/OpenSearch-2.x%20%7C%203.x-orange?logo=opensearch)](https://opensearch.org)

> 🔗 **RFC**: [opensearch-project/.github#491](https://github.com/opensearch-project/.github/issues/491)
> · 🎯 **Live Demo**: [f5cmk2hxwx.us-west-2.awsapprunner.com](https://f5cmk2hxwx.us-west-2.awsapprunner.com)
> · 📖 **Docs**: [docs/](docs/)

**Try it now:**
```bash
# Health check
curl https://f5cmk2hxwx.us-west-2.awsapprunner.com/health

# Get a scoped token
curl -X POST https://f5cmk2hxwx.us-west-2.awsapprunner.com/oauth/token \
  -d "grant_type=client_credentials&client_id=demo-agent&client_secret=demo-secret&scope=read:logs-*"
```

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
# Start proxy + OpenSearch + Keycloak
docker compose up

# Get a scoped token
curl -X POST http://localhost:8443/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=my-agent" \
  -d "client_secret=secret" \
  -d "scope=read:logs-*"

# Query OpenSearch through the proxy
curl -H "Authorization: Bearer <token>" \
  http://localhost:8443/logs-*/_search \
  -d '{"query": {"match": {"level": "error"}}}'

# Revoke when done
curl -X DELETE http://localhost:8443/oauth/token/<token-id>
```

Or use the CLI:

```bash
oauth4os login --provider keycloak
oauth4os create-token --scope "read:logs-*" --name my-agent
oauth4os status
oauth4os revoke <token-id>
```

## Architecture

```
┌─────────────┐     ┌──────────────────────────┐     ┌─────────────────┐
│  Clients    │     │      oauth4os proxy       │     │   OpenSearch    │
│             │     │      (Go, :8443)          │     │                 │
│  AI Agent   │────▶│  ┌─────────────────────┐  │────▶│  Engine :9200   │
│  CI/CD      │     │  │ JWT Validation      │  │     │                 │
│  Slack Bot  │     │  │ Scope → Role Mapping│  │────▶│  Dashboards     │
│  CLI        │     │  │ Cedar Policies      │  │     │  :5601          │
│  MCP Server │     │  │ Rate Limiting       │  │     │                 │
│  Browser    │     │  │ Audit Logging       │  │     │  Security plugin│
│             │     │  │ Token Introspection │  │     │  unchanged      │
└─────────────┘     │  └─────────────────────┘  │     └─────────────────┘
                    └────────────┬───────────────┘
                                │
                    ┌───────────▼────────────┐
                    │    OIDC Provider       │
                    │  Keycloak · Auth0      │
                    │  Okta · Dex · Google   │
                    └────────────────────────┘
```

## Features

### Core Proxy
- **JWT validation** — JWKS auto-discovery, RS256/ES256, configurable clock skew
- **Scope-to-role mapping** — `read:logs-*` → OpenSearch `logs_reader` role
- **Unified auth** — single entry point for Engine + Dashboards APIs
- **Rate limiting** — per-client token bucket, configurable RPM per scope, `429 + Retry-After`
- **Request tracing** — `X-Request-ID` on every proxied request
- **Connection pooling** — configurable idle connections, timeouts
- **Graceful shutdown** — drain in-flight requests on SIGTERM
- **Prometheus metrics** — `/metrics` endpoint (requests, auth, Cedar, upstream errors)
- **Health check** — `/health` with version and uptime

### Auth & Security
- **Token lifecycle** — issue, refresh, revoke, list, inspect via REST API
- **Token introspection** — RFC 7662 compliant endpoint
- **PKCE flow** — secure browser-based auth (RFC 7636)
- **Cedar policies** — fine-grained access control, index-level deny rules
- **Multi-tenancy** — per-OIDC-provider scope mapping and Cedar policies
- **Audit logging** — structured JSON logs for every proxied request
- **Any OIDC provider** — Keycloak, Auth0, Okta, Dex, Google

### Developer Experience
- **CLI tool** — `oauth4os login`, `create-token`, `revoke`, `status`
- **OSD plugin** — token management UI in OpenSearch Dashboards (list, create, revoke)
- **MCP server** — reference integration for AI agents (search, create index, mappings, aggregations)
- **Zero breaking changes** — existing auth methods continue to work

## OSD Plugin — Token Management

The OpenSearch Dashboards plugin provides a UI for managing OAuth tokens:

| Feature | Description |
|---------|-------------|
| **List tokens** | Table with status badges, scope tags, time-ago, copy-to-clipboard |
| **Create token** | Client credentials form with scope help text |
| **Revoke token** | Confirmation dialog, immediate invalidation |
| **Token result** | Copy buttons for access + refresh tokens |

Located at `plugins/oauth4os-dashboards/`. Uses EUI components (EuiBasicTable, EuiModal, EuiConfirmModal, EuiCopy, EuiBadge).

## MCP Server — AI Agent Integration

Reference MCP server for AI agents to query OpenSearch securely:

```bash
cd examples/mcp-server
pip install -r requirements.txt
python server.py
```

Tools: `search_logs`, `create_index`, `delete_docs`, `get_mappings`, `aggregate`.

## Project Structure

```
cmd/
  proxy/              — Main proxy binary
  cli/                — CLI tool (login, create-token, revoke, status)
internal/
  jwt/                — JWT validation + JWKS cache
  scope/              — Scope-to-role mapping engine
  cedar/              — Cedar policy evaluation + multi-tenant
  token/              — Token lifecycle (issue/refresh/revoke/list)
  introspect/         — RFC 7662 token introspection
  pkce/               — PKCE authorization flow
  ratelimit/          — Per-client token bucket rate limiter
  config/             — YAML config loader
  audit/              — Structured request audit logging
plugins/
  oauth4os-dashboards/ — OSD plugin (TypeScript/React)
examples/
  mcp-server/         — MCP server reference (Python)
deploy/
  cdk/                — AWS CDK stack
  helm/               — Helm chart (oauth4os/)
  keycloak/           — Keycloak realm export for dev
web/                  — Landing page (demo site)
bench/                — Go benchmarks
test/
  integration/        — Integration tests (proxy, scope, Cedar)
  e2e/                — End-to-end tests (Docker-based)
docs/                 — Architecture, security, benchmarks, testing
```

## Configuration

```yaml
# config.yaml
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

rate_limit:
  default_rpm: 60
  per_scope:
    "read:logs-*": 120
    "admin": 30

listen: :8443

tls:
  enabled: false
  cert_file: /etc/oauth4os/tls.crt
  key_file: /etc/oauth4os/tls.key
```

## Deployment

### Docker

```bash
docker compose up                    # Dev: proxy + OpenSearch + Keycloak
docker compose -f docker-compose.demo.yml up  # Demo mode
```

### Helm

```bash
helm install oauth4os deploy/helm/oauth4os/ \
  --set config.upstream.engine=https://opensearch:9200
```

### AWS CDK

```bash
cd deploy/cdk && pip install -r requirements.txt
cdk deploy
```

### Binary

Download from [Releases](https://github.com/seraphjiang/oauth4os/releases) — linux/mac/windows, amd64/arm64.

```bash
oauth4os-proxy --config config.yaml
```

## API Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `POST /oauth/token` | POST | Issue token (client_credentials or refresh_token grant) |
| `GET /oauth/tokens` | GET | List active tokens |
| `GET /oauth/token/{id}` | GET | Get token details |
| `DELETE /oauth/token/{id}` | DELETE | Revoke token |
| `POST /oauth/introspect` | POST | Token introspection (RFC 7662) |
| `GET /oauth/authorize` | GET | PKCE authorization (browser flow) |
| `GET /health` | GET | Health check + version |
| `GET /metrics` | GET | Prometheus metrics |
| `/*` | ANY | Reverse proxy to OpenSearch (with auth) |

## Roadmap

| Phase | Scope | Status |
|-------|-------|--------|
| Phase 1 | OAuth proxy MVP — JWT, scope mapping, token lifecycle, CLI, Docker | ✅ Shipped |
| Phase 2 | OSD plugin, rate limiting, introspection, PKCE, multi-tenancy, MCP server | ✅ Shipped |
| Phase 3 | Production polish — docs, benchmarks, CI/CD, security hardening | 🔨 Building |

## Documentation

| Doc | Description |
|-----|-------------|
| [docs/architecture.md](docs/architecture.md) | Architecture, data flow, component descriptions |
| [docs/security.md](docs/security.md) | Threat model, auth flows, JWT validation |
| [docs/quickstart.md](docs/quickstart.md) | Step-by-step setup guide |
| [docs/user-manual.md](docs/user-manual.md) | Complete user manual |
| [docs/benchmarks.md](docs/benchmarks.md) | Performance numbers, scaling guidance |
| [docs/testing.md](docs/testing.md) | How to run tests, coverage targets |

## Contributing

Pull requests welcome. Please open an issue first to discuss major changes.

1. Fork the repo
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Run tests: `make test` or `go test ./...`
4. Commit following [Conventional Commits](https://www.conventionalcommits.org/)
5. Open a pull request

## License

Apache 2.0 — see [LICENSE](LICENSE).
