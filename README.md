# oauth4os — OAuth 2.0 Proxy for OpenSearch

**Secure machine-to-machine access for OpenSearch.** OAuth 2.0 proxy that validates JWT tokens, maps scopes to OpenSearch security roles, and forwards requests to both Engine and Dashboards — with zero changes to existing components.

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22-00ADD8)](https://go.dev)

> 🔗 **RFC**: [opensearch-project/.github#491](https://github.com/opensearch-project/.github/issues/491)

> 🎯 **Demo**: [oauth4os.huanji.profile.aws.dev](http://oauth4os.huanji.profile.aws.dev/)

---

## Why?

OpenSearch has OIDC auth and API Keys (3.7), but lacks the developer experience layer:

| | Grafana | Datadog | Elastic | **OpenSearch** |
|---|---|---|---|---|
| OIDC/SSO | ✅ | ✅ | ✅ | ✅ |
| API Keys | ✅ | ✅ | ✅ | 🔄 (3.7) |
| OAuth Apps / Scoped Tokens | ✅ | ✅ | ✅ | **❌** |
| Token Governance UI | ✅ | ✅ | ✅ | **❌** |

oauth4os fills this gap — scoped tokens, OIDC federation, unified auth across Engine + Dashboards, governance UI.

## Quick Start

```bash
docker compose up
```

```bash
# Get a scoped token
curl -X POST http://localhost:8443/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=my-agent" \
  -d "client_secret=secret" \
  -d "scope=read:logs-*"

# Use it
curl -H "Authorization: Bearer <token>" \
  http://localhost:8443/logs-*/_search \
  -d '{"query": {"match": {"level": "error"}}}'
```

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  AI Agent   │────▶│  OAuth Proxy │────▶│   OpenSearch     │
│  CI/CD      │     │  (Go, :8443) │     │   Engine (:9200) │
│  Slack Bot  │     │              │────▶│   Dashboards     │
│  CLI        │     │  JWT + JWKS  │     │   (:5601)        │
└─────────────┘     └──────┬───────┘     └─────────────────┘
                           │
                    ┌──────▼───────┐
                    │ OIDC Provider│
                    │ Keycloak     │
                    │ Auth0 / Okta │
                    └──────────────┘
```

## Features

- **JWT validation** — JWKS auto-discovery, RS256/ES256
- **Scope-to-role mapping** — `read:logs-*` → OpenSearch `logs_reader` role
- **Unified auth** — single entry point for Engine + Dashboards APIs
- **Token lifecycle** — issue, revoke, list, inspect via REST API
- **Cedar policies** — fine-grained access control (Phase 3)
- **CLI tool** — `oauth4os login`, `oauth4os create-token`, `oauth4os revoke`
- **Zero breaking changes** — existing auth methods continue to work
- **Any OIDC provider** — Keycloak, Auth0, Okta, Dex

## Project Structure

```
cmd/
  proxy/          — Main proxy binary
  cli/            — CLI tool (oauth4os)
internal/
  proxy/          — HTTP proxy + routing
  jwt/            — JWT validation + JWKS cache
  scope/          — Scope-to-role mapping engine
  cedar/          — Cedar policy evaluation
  config/         — YAML config loader
  audit/          — Request audit logging
  health/         — Health check endpoints
  token/          — Token lifecycle (issue/revoke/list)
api/
  openapi.yaml    — OpenAPI 3.0 spec
deploy/
  docker/         — Dockerfile + docker-compose
  helm/           — Helm chart
  cdk/            — AWS CDK stack
docs/
  architecture.md
  configuration.md
  deployment.md
test/
  integration/    — Integration tests with real OpenSearch
  e2e/            — End-to-end tests
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

listen: :8443
```

## Phases

| Phase | Scope | Status |
|-------|-------|--------|
| Phase 1 | OAuth proxy MVP — JWT, scope mapping, CLI, Docker | 🔨 Building |
| Phase 2 | OSD plugin — token management UI, consent screen | Planned |
| Phase 3 | Cedar policies — fine-grained local policy evaluation | Planned |

## License

Apache 2.0
