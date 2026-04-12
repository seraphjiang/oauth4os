# Changelog

All notable changes to oauth4os are documented here.
Format based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### Phase 5 — RFC Readiness
#### Added
- CONTRIBUTING.md with dev setup, PR guidelines, project structure
- CHANGELOG.md (this file)
- GitHub Pages workflow for docs publishing
- Monitoring stack: docker-compose.monitoring.yml with Prometheus + Grafana
- Pre-built Grafana dashboard (8 panels: req/s, auth, errors, Cedar, rate limits)
- RFC response draft (docs/rfc-response.md)
- Go SDK client library (pkg/client/)
- OpenAPI 3.0 spec (docs/api-spec.yaml)
- Real-world examples: Fluent Bit, LangChain agent, K8s sidecar
- Comprehensive test report (docs/test-report.md)

### Phase 4 — Backlog & Hardening
#### Added
- Prometheus alerting rules (7 pre-built alerts)
- Structured JSON logger (internal/logging/)
- OpenTelemetry tracing for proxy stages (internal/tracing/)
- Security scanning workflows (gosec + govulncheck + CodeQL)
- Chaos tests: kill upstream, expired JWKS, clock skew
- Token exchange (RFC 8693)
- Dynamic client registration (RFC 7591)
- Audience validation per provider
- Concurrent token revocation race condition tests
- Migration guide and comparison doc

#### Fixed
- PKCE open redirect vulnerability (P0)
- Header injection in proxied requests
- Error responses no longer leak internal details
- Request body size limit to prevent DoS

### Phase 3 — Polish & Docs
#### Added
- Dependabot config (Go, Docker, GitHub Actions)
- CodeQL security scanning workflow
- docker-compose.test.yml with healthchecks for CI
- Goreleaser config for automated releases
- Makefile targets: build, build-all, lint, docker, release, clean
- CODEOWNERS file
- Architecture doc with Mermaid diagrams
- Security doc with threat model
- User manual (18 sections)
- Shell completions for CLI (bash/zsh/fish)

### Phase 2 — Feature Expansion
#### Added
- GitHub release workflow (6 platforms + multi-arch Docker to ghcr.io)
- CI pipeline: lint, test, build, Docker, integration (5 jobs)
- Helm chart with deployment, service, configmap, ingress
- Rate limiting with token bucket (per-client, configurable)
- OSD plugin scaffolding for token management UI
- Token introspection endpoint (RFC 7662)
- PKCE flow for browser clients
- Multi-tenancy per OIDC provider
- Go benchmarks for JWT, scope, Cedar, proxy round-trip

### Phase 1 — MVP
#### Added
- OAuth 2.0 reverse proxy for OpenSearch Engine + Dashboards
- JWT validation with OIDC auto-discovery and JWKS caching
- Scope-to-role mapping (configurable in YAML)
- Cedar policy engine with permit/forbid and glob matching
- Token manager: issue, revoke, list client credentials tokens
- Audit logging to stdout (JSON)
- CLI tool: login, create-token, revoke-token, status
- MCP server example (7 tools for AI agent integration)
- CDK stack: OpenSearch + ECS Fargate + ALB + Route53
- Landing page for demo site
- Docker + docker-compose for local development
- 10 integration tests, 9 E2E tests

[Unreleased]: https://github.com/seraphjiang/oauth4os/compare/main...HEAD
