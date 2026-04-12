# Changelog

All notable changes to oauth4os are documented here.
Format based on [Keep a Changelog](https://keepachangelog.com/).

## [v0.5.0] — 2026-04-12

### Added
- Request retry with exponential backoff for upstream 5xx (3 retries, 100ms base)
- Background upstream health check (30s interval, status in /health/deep)
- Response cache for GET requests (5s TTL, 1000 entries, X-Cache header)
- Circuit breaker state + upstream latency in /health/deep
- Load shedding middleware (503 when active connections > threshold)
- Request timeout middleware (30s default, 504 on deadline)
- Idempotency-Key request deduplication for POST/PUT/PATCH
- OTLP span exporter with batched HTTP delivery
- Webhook event notifications for token lifecycle
- Prometheus alerting rules (8 rules for all 14 metrics)
- Docker Compose monitoring stack (Prometheus + Grafana)
- Canary deployment guide (AppRunner, K8s, Route 53)
- Helm chart integration test script (kind cluster)
- Comparison benchmarks: cache +75ns, circuit breaker +47ns overhead
- Resilience chain integration tests (retry → circuit → cache)
- 5 new Prometheus metrics: cache_hits, cache_misses, circuit_opens, upstream_latency_ms, upstream_healthy

### Fixed
- URL parse errors in main.go now fail fast instead of silent ignore
- Removed duplicate loadshed/guard.go conflicting with shedder.go
- Removed redundant _ = suppressions

## [v0.4.0] — 2026-04-12

### Added
- Device flow (RFC 8628) for CLI/IoT devices
- Token binding (DPoP prep, RFC 9449)
- Client authentication: client_secret_post, client_secret_basic, private_key_jwt
- SQL query support in CLI
- Query history and bookmarks in CLI
- Shell completions (bash/zsh)
- PWA manifest + service worker for demo app
- Keyboard shortcuts in demo app (/, j/k, r)
- Dark/light theme toggle
- Loading skeletons for all pages
- Performance docs, FAQ, SDK guide
- Property-based tests for SigV4 signing
- Fuzz tests for JSON parsing paths
- Stress test: 232 req/s throughput

### Fixed
- Race condition in token refresh family tracking

## [v0.3.0] — 2026-04-12

### Added
- Developer portal with OAuth app management UI
- 9-command CLI demo (login, search, services, indices, tail, config, alias, history, completion)
- curl-pipe installer (GET /install.sh)
- CORS middleware with configurable origins
- Rate limit headers (X-RateLimit-Limit/Remaining/Reset)
- Request ID propagation in all error responses and logs
- Graceful shutdown (drain connections, save clients, flush audit)
- Token inspector page (/developer/token-inspector)
- Health dashboard (/admin/health)
- Architecture and deployment docs

### Fixed
- go vet warnings across entire codebase (7 instances)

## [v0.2.0] — 2026-04-12

### Added
- Demo web app at /demo/app (PKCE login, log search, filters)
- Client persistence to data/clients.json (atomic writes, backup, corruption recovery)
- Sliding window token refresh with family revocation
- Client CRUD endpoints (LIST/UPDATE/DELETE + secret rotation)
- SigV4 signing for AWS OpenSearch Serverless (AOSS)
- Seed script for demo data (820+ log entries across 5 services)
- Live data simulator (10 entries/minute)
- Auto-deploy pipeline (ECR push → AppRunner)
- Swagger UI and developer analytics pages extracted to go:embed

### Fixed
- Backup handler wired to mux (was discarded with _ =)
- Config validation for AOSS endpoint format and SigV4 region

## [v0.1.0] — 2026-04-12

### Added
- OAuth 2.0 reverse proxy for OpenSearch (Go, 2 dependencies)
- JWT validation with OIDC auto-discovery and JWKS caching
- Scope-to-role mapping (YAML config)
- Cedar policy engine (permit/forbid, glob matching, when/unless conditions)
- Token manager: issue, revoke, refresh, list (client credentials)
- PKCE flow for browser clients (RFC 7636)
- Token introspection (RFC 7662)
- Dynamic client registration (RFC 7591)
- Token exchange (RFC 8693)
- Multi-tenancy per OIDC provider
- Rate limiting with token bucket (per-client)
- IP allowlist/denylist per client
- Mutual TLS client authentication
- Webhook external authorizer
- Session management with force logout
- Audit logging (structured JSON)
- OpenTelemetry tracing for proxy stages
- CLI tool (login, create-token, revoke-token, status)
- Terraform provider (clients, scopes, Cedar policies)
- Grafana datasource plugin
- GitHub Action for CI/CD token issuance
- Cedar Policy Playground (web)
- 6 deployment examples (Terraform ECS, Lambda, Fluent Bit, K8s sidecar, LangChain, MCP server)
- Helm chart, CDK stack, Docker Compose
- OpenAPI 3.0 spec (30 endpoints)
- 10 ADRs, security scan, benchmarks doc
- CONTRIBUTING.md, SECURITY.md
- 24 test packages, 371 tests passing

### Security
- Fixed PKCE open redirect vulnerability (P0)
- Fixed header injection in proxied requests
- Error responses no longer leak internal details

[v0.5.0]: https://github.com/seraphjiang/oauth4os/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/seraphjiang/oauth4os/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/seraphjiang/oauth4os/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/seraphjiang/oauth4os/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/seraphjiang/oauth4os/releases/tag/v0.1.0
