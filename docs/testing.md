# Testing Guide

How to run tests, what they cover, and how CI works.

---

## Quick Start

```bash
# Unit tests (no external dependencies)
make test-unit

# Integration tests (needs docker compose)
docker compose -f docker-compose.test.yml up -d --wait
go test -v -tags=integration ./test/integration/...
docker compose -f docker-compose.test.yml down -v

# E2E tests (needs full demo stack)
make demo-up
make test-e2e        # bash script
make test-e2e-go     # Go tests
make demo-down

# Benchmarks
go test -bench=. -benchmem ./bench/...
```

---

## Test Levels

### Unit Tests — `./internal/...`

72 test functions across all 9 internal packages. No external dependencies — all I/O is mocked.

| Package | Tests | What's Covered |
|---|---|---|
| `token` | 14 | Issuance, client auth, scope validation, refresh flow, revocation, listing, lookup |
| `jwt` | 12 | Malformed tokens, scope extraction, JWKS key lookup, provider matching |
| `cedar` | 10 | Policy evaluation, glob matching, permit/forbid, conditions, multi-tenant |
| `ratelimit` | 9 | Token bucket refill, per-scope RPM, 429 + Retry-After, middleware |
| `introspect` | 9 | RFC 7662 active/inactive, adapter for all token states, method check |
| `pkce` | 9 | Authorize, exchange, bad verifier, code reuse, redirect mismatch, cleanup |
| `config` | 4 | Valid config, missing file, invalid YAML, tenants + rate_limits |
| `scope` | 3 | Global mapping, tenant override, role deduplication |
| `audit` | 2 | Log output format |

Run with race detector:
```bash
go test -race -coverprofile=coverage.out ./internal/...
go tool cover -func=coverage.out | tail -1
```

### Integration Tests — `./test/integration/`

26 test functions. Require a running proxy + OpenSearch (via `docker-compose.test.yml`).

| File | Tests | What's Covered |
|---|---|---|
| `proxy_test.go` | 11 | Health check, token CRUD, proxy passthrough, bearer auth, invalid token |
| `scope_test.go` | 10 | Scope enforcement, token expiry, double revoke, routing, concurrency |
| `cedar_test.go` | 5 | Permit/forbid with real policies, conditions, multi-provider |

The integration tests start a real proxy (via `TestMain`) and issue real HTTP requests.

```bash
docker compose -f docker-compose.test.yml up -d --wait
go test -v -tags=integration ./test/integration/...
```

### E2E Tests — `./test/e2e/`

Full stack: Keycloak (OIDC provider) → oauth4os proxy → OpenSearch.

**Bash script** (`test/e2e/run.sh`) — 12 test cases:
- Token issuance via proxy (log-reader, admin-agent)
- Invalid credentials rejection
- Scope enforcement: admin creates index, reader searches, unauth rejected
- Token revocation
- Audit trail, health endpoint, token listing
- Cleanup

**Go tests** (`test/e2e/e2e_test.go`) — 8 test functions:
- Same coverage, configurable via `PROXY_URL`, `KEYCLOAK_URL`, `OPENSEARCH_URL` env vars

```bash
make demo-up
make test-e2e       # bash
make test-e2e-go    # Go
make demo-down
```

### Benchmarks — `./bench/`

10 benchmarks for performance-critical paths:

| Benchmark | What it measures |
|---|---|
| `BenchmarkScopeMapper_1Scope` | Single scope resolution |
| `BenchmarkScopeMapper_5Scopes` | Multi-scope resolution |
| `BenchmarkScopeMapper_Miss` | No matching scope |
| `BenchmarkCedar_1Policy` | Single policy evaluation |
| `BenchmarkCedar_10Policies` | 10-policy evaluation |
| `BenchmarkCedar_100Policies` | 100-policy evaluation |
| `BenchmarkCedar_DenyAll` | All-deny policy set |
| `BenchmarkCedar_ForbidOverride` | Forbid overriding permit |
| `BenchmarkProxyRoundTrip` | Full proxy request with auth |
| `BenchmarkProxyRoundTrip_Passthrough` | Proxy without auth |

```bash
go test -bench=. -benchmem -count=5 ./bench/...
```

---

## CI Pipeline

Defined in `.github/workflows/ci.yml`. Runs on every push to `main` and every PR.

```
lint ──→ test ──→ docker
    └──→ build ──→ integration
```

| Job | What it does | Duration |
|---|---|---|
| `lint` | golangci-lint + go vet | ~30s |
| `test` | Unit tests with `-race` + coverage report | ~15s |
| `build` | Build proxy + CLI binaries | ~10s |
| `docker` | Build Docker image + smoke test (health check) | ~60s |
| `integration` | docker-compose.test.yml up → integration tests → teardown | ~90s |

### What's NOT in CI

- **E2E tests** — require docker-compose.demo.yml (Keycloak + OpenSearch Dashboards). Too heavy for CI. Run manually before releases.
- **Benchmarks** — run locally for performance regression checks.

---

## Coverage Targets

| Level | Current | Target |
|---|---|---|
| Unit test functions | 72 | Maintain ≥70 |
| Integration test functions | 26 | Maintain ≥25 |
| E2E test cases | 20 | Maintain ≥15 |
| Line coverage (unit) | TBD | ≥80% |
| Packages with tests | 9/9 | 100% |

---

## Adding Tests

### Unit test conventions
- File: `internal/<pkg>/<name>_test.go`
- Package: same as source (e.g., `package token`)
- Use `httptest.NewRecorder()` + `httptest.NewRequest()` for HTTP handlers
- Mock external dependencies — no network calls

### Integration test conventions
- File: `test/integration/<name>_test.go`
- Build tag: `//go:build integration`
- Use `TestMain` for setup/teardown
- Tests run against real proxy at `http://localhost:8443`

### E2E test conventions
- Go tests: `test/e2e/e2e_test.go`, configurable via env vars
- Bash tests: `test/e2e/run.sh`, self-contained with pass/fail reporting

---

## Test Count Summary

| Level | Count |
|---|---|
| Unit tests | 72 |
| Integration tests | 26 |
| E2E tests (Go) | 8 |
| E2E tests (Bash) | 12 |
| Other (proxy) | 4 |
| Benchmarks | 10 |
| **Total** | **132** |
