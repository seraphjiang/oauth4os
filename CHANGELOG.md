# Changelog

All notable changes to oauth4os are documented here.
Format based on [Keep a Changelog](https://keepachangelog.com/).

## [v1.0.0] — 2026-04-12 🎉 Production Ready

### Added
- Gzip response compression middleware (sync.Pool, 28µs/op)
- Toast notifications + offline detection in demo app
- `--json` output flag on all CLI data commands (fully scriptable)
- mTLS + keyring mutation tests (117 total mutations killed)
- Compress edge tests + benchmarks (pool safety, concurrent)
- Traceparent benchmarks (80ns/op, 0 allocs)
- SigV4 benchmarks (6µs/op), token lookup benchmarks (18ns/op)
- Registration edge tests, tokenbind + discovery mutation tests
- 726 tests across 50 packages

### Fixed
- Race condition in token exchange StaticTokenIssuer (added mutex)
- 3 E2E test failures (consent flow, health/deep 503 acceptance)
- Non-TTY output handling in CLI stats command
- Code quality audit: zero TODOs, zero vet warnings, zero duplicate routes

### Stats
- 542 commits, 443 files, 726 tests, 117 mutations killed, 50 packages
- Docker Hub: `docker run -p 8443:8443 jianghuan/oauth4os:latest`
- Live demo: https://f5cmk2hxwx.us-west-2.awsapprunner.com

## [v0.8.0] — 2026-04-12

### Added
- Security headers middleware: HSTS, X-Content-Type-Options nosniff, X-Frame-Options DENY
- CLI `latency` command — 26 total CLI commands, all 47 race-clean
- Setup wizard (3-step with live checks) + Open Graph meta tags
- CLI `alerts` command + LoadShedding Prometheus alert rule
- Changelog web page, sitemap (13 URLs), full link audit
- Operator runbook — incidents, token ops, scaling
- Bytes in/out metrics, ADR-014 W3C Traceparent, ADR-015 Multi-Tier Rate Limiting
- Grafana dashboard: all 16 Prometheus metrics (cache, circuit breaker, loadshed, upstream)
- Man page updated for all 26 commands
- Fuzz tests: config (101K executions), client store (7K executions), 0 panics
- Mutation tests: 87 mutations killed (accesslog, events, CORS, compress, healthcheck, retry, webhook, timeout, CIBA)
- 771 tests across 47 packages

### Fixed
- CRITICAL: Cedar policy engine data race — added RWMutex on hot path
- Timeout middleware data race — atomic.Bool for written flag
- Dockerfile GOFLAGS=-insecure → GONOSUMCHECK + GOINSECURE
- Referrer policy on 12 pages

### Performance
- Cedar permit: 110ns, introspection cache hit: 63ns

## [v0.7.0] — 2026-04-12

### Added
- Docker Hub publishing: `docker run -p 8443:8443 jianghuan/oauth4os:latest`
- /ready readiness probe with shutdown flag for graceful deploys
- /version endpoint with build metadata
- Mutation testing: 51 mutations killed across Cedar, JWT, loadshed, idempotency, DPoP, PAR
- DPoP + PAR mutation tests
- CSV audit export + WCAG skip-to-content accessibility
- Keyboard shortcuts modal + favicon across all pages
- 558 tests across 46 packages

### Fixed
- Token binding was hashing ephemeral port — completely ineffective (fixed)
- JWT test discovery response missing issuer field
- Retry transport vet warning

## [v0.6.0] — 2026-04-12

### Added
- CIBA (Client-Initiated Backchannel Authentication) flow
- PAR (Pushed Authorization Requests) endpoint
- Config UI page (/admin/config) — edit config from browser
- Comprehensive CLI guide (23 commands)
- OAuth flows documentation with Mermaid diagrams (8 flows)
- Frontend contributing guide
- Consistent navigation across all demo pages
- Cedar CRUD tests (coverage 78% → 87%)
- SigV4 regression tests for 3 bugs fixed earlier
- 514 tests, 22 fuzz targets

### Fixed
- Duplicate route crash: /admin/config and /admin/backup conflicted with admin API
- backupHandler.Register and configUI.Register removed from main.go (crash fix)

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

[v0.7.0]: https://github.com/seraphjiang/oauth4os/compare/v0.6.0...v0.7.0
[v0.6.0]: https://github.com/seraphjiang/oauth4os/compare/v0.5.0...v0.6.0
[v1.0.0]: https://github.com/seraphjiang/oauth4os/compare/v0.8.0...v1.0.0
[v0.8.0]: https://github.com/seraphjiang/oauth4os/compare/v0.7.0...v0.8.0
[v0.7.0]: https://github.com/seraphjiang/oauth4os/compare/v0.6.0...v0.7.0
[v0.6.0]: https://github.com/seraphjiang/oauth4os/compare/v0.5.0...v0.6.0
[v0.5.0]: https://github.com/seraphjiang/oauth4os/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/seraphjiang/oauth4os/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/seraphjiang/oauth4os/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/seraphjiang/oauth4os/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/seraphjiang/oauth4os/releases/tag/v0.1.0
