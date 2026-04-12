# v1.1.0 Roadmap

## Release Goals

v1.1.0 focuses on **production hardening** and **developer experience**. v1.0.0 proved the architecture works; v1.1.0 makes it deployable by any team without hand-holding.

---

## Planned Features

### 1. Token Refresh Flow
**Status**: In progress (auth-eng)
**What**: `POST /oauth/token` with `grant_type=refresh_token`
**Why**: Clients currently re-authenticate when tokens expire. Refresh tokens allow seamless session continuity.
**Note**: Token manager already has refresh token storage (`internal/token/manager.go:134`). Needs endpoint wiring and rotation policy.

### 2. Integration Tests
**Status**: In progress (sde)
**What**: `test/integration/` — spin up proxy, hit every endpoint, verify responses
**Why**: Unit tests cover 47 packages at ≥80%, but no end-to-end verification that the assembled proxy works correctly.

### 3. CLI Login Command
**Status**: In progress (query-eng)
**What**: `oauth4os-demo login` — interactive PKCE flow from terminal
**Why**: Currently requires manual token acquisition. CLI login opens browser, catches callback on localhost, stores token for subsequent commands.

### 4. Request Latency Histograms
**Status**: In progress (index-eng)
**What**: Prometheus histogram for request latency by endpoint
**Why**: Currently only counters. Histograms enable p50/p95/p99 latency monitoring and SLO alerting.

### 5. Docker Compose
**Status**: In progress (frontend)
**What**: `docker-compose.yml` at repo root that just works on Mac and Linux
**Why**: Current compose file has build issues on Mac. One-command local development setup.

### 6. Automated Releases
**Status**: In progress (devops)
**What**: goreleaser config for automated binary + Docker builds on tag
**Why**: Manual Docker builds don't scale. Tag → build → push should be fully automated.

### 7. Mutation Test Coverage
**Status**: In progress (test)
**What**: Mutation tests for every exported function
**Why**: 202 mutations killed so far. Mutation testing catches bugs that line coverage misses.

---

## Production Readiness Audit

What's needed for a real production deployment beyond what v1.0.0 provides:

### ✅ Already Handled
- TLS termination (config supports cert/key files)
- Graceful shutdown (30s drain, `/ready` probe)
- Rate limiting (per-scope, configurable)
- Circuit breaker (automatic upstream failure detection)
- Load shedding (CPU-based admission control)
- Structured audit logging
- Prometheus metrics (16 metrics)
- OTLP tracing (W3C Traceparent)
- Health checks (liveness, deep, readiness)
- CORS (configurable origins)
- Security headers (HSTS, X-Content-Type-Options, X-Frame-Options)

### ⚠️ Gaps to Address

#### ✅ Secret Management (Closed)
**Current**: Secrets in YAML config file or environment variables.
**Needed**: Integration with AWS Secrets Manager, HashiCorp Vault, or Kubernetes secrets.
**Recommendation**: Add `secret_ref` field in config that resolves at startup:
```yaml
providers:
  - name: keycloak
    client_secret: !secret aws:secretsmanager:oauth4os/keycloak-secret
```

#### ✅ Key Rotation Automation (Closed)
**Current**: Keyring rotates signing keys automatically (configurable interval), but JWKS consumers must poll.
**Needed**: 
- Configurable rotation schedule (default: 24h)
- Grace period where old key still validates (default: 48h)
- JWKS endpoint already serves all active keys — this works today
- Add admin endpoint to trigger manual rotation: `POST /admin/keys/rotate`

#### ✅ TLS Certificate Renewal (Closed)
**Current**: Static cert/key files. Restart required to pick up new certs.
**Needed**: Automatic cert reload on file change (fsnotify) or ACME/Let's Encrypt integration.
**Workaround**: Use a reverse proxy (nginx, Caddy, ALB) for TLS termination — recommended for production anyway.

#### ✅ Persistent Token Store (Closed)
**Current**: In-memory token store. Tokens lost on restart.
**Needed**: Optional DynamoDB/Redis/PostgreSQL backend for token persistence.
**Workaround**: Short-lived tokens (15min) with refresh tokens. Restart only affects active sessions.
**ADR**: [006-in-memory-token-store.md](adr/006-in-memory-token-store.md) documents this as intentional for MVP.

#### Multi-Instance Coordination (Priority: Low)
**Current**: Stateless — each instance has its own token store and cache.
**Impact**: Token revocation only affects the instance that received the revoke request.
**Needed**: Shared revocation list (Redis pub/sub or DynamoDB stream).
**Workaround**: Short token TTLs (15min) limit the window of revocation inconsistency.

#### Backup Encryption (Priority: Low)
**Current**: Config backup/restore exports plaintext JSON.
**Needed**: Encrypt backup with KMS or age before export. Decrypt on import.

---

## Timeline

| Week | Milestone |
|------|-----------|
| 1 | v1.1.0-rc1: refresh tokens, integration tests, CLI login, histograms |
| 2 | v1.1.0-rc2: docker compose, goreleaser, mutation coverage complete |
| 3 | v1.1.0: release with all features, updated docs |
| 4+ | v1.2.0 planning: secret management, persistent store, cert reload |

---

## Success Criteria

- [ ] `docker compose up` works on Mac and Linux without edits
- [ ] Integration tests cover all 39+ endpoints
- [ ] `oauth4os-demo login` completes PKCE flow from terminal
- [ ] Prometheus histograms for p50/p95/p99 latency
- [ ] goreleaser produces binaries for linux/darwin/windows amd64/arm64
- [ ] Every exported function has a mutation test
- [ ] Token refresh flow works end-to-end
- [ ] All production gaps documented with workarounds
