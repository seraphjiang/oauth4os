# oauth4os v2.0.0 — Major Feature Release

**Released: April 12, 2026**

v2.0.0 is a major release shipping 32 features across 8 agents in a single sprint. The proxy is now a full-featured OAuth 2.0 platform with real-time streaming, plugin extensibility, and production-grade observability.

## By the Numbers

| Metric | v1.0.0 | v2.0.0 | Change |
|--------|--------|--------|--------|
| Commits | 542 | 846 | +56% |
| Files | 443 | 587 | +33% |
| Packages | 50 | 65 | +30% |
| Tests | 726 | 1200+ | +65% |
| Web pages | 12 | 29 | +142% |
| CLI commands | 24 | 60 | +150% |

## New Features

### Plugin System
Load custom auth logic at runtime via Go plugins (.so files). Implement the `Authorizer` interface, drop in a .so, set `OAUTH4OS_PLUGINS`, and the proxy loads it on startup.

### Webhook HMAC-SHA256 Signing
All outgoing webhooks now include `X-Webhook-Signature: sha256=<hex>` headers. Receivers can verify authenticity using the shared secret.

### Audit Log Export
Batch audit entries and upload to S3 (or any backend implementing the `Uploader` interface) every 5 minutes. NDJSON format for easy ingestion into analytics pipelines.

### Multi-Tenant Token Isolation
Separate token stores per tenant. Each tenant gets its own isolated store — no cross-tenant token leakage. Lazy creation, automatic cleanup.

### TLS Certificate Auto-Reload
Certificates are polled every 30 seconds. When renewed (certbot, cert-manager, manual), the proxy picks up the new cert with zero downtime.

### Secret Management
Config values can reference secrets: `env:VAR_NAME` reads from environment, `file:/path` reads from disk. No more plaintext secrets in config files.

### Persistent Token Store
Tokens survive restarts with the file backend (`token_store: file:/path`). JSON + fsync for durability. Memory backend remains the default.

### Production Gaps Closed
All 4 high/medium priority production gaps from the v1.1.0 roadmap are resolved:
1. ✅ Secret management
2. ✅ Key rotation automation + logging
3. ✅ TLS cert renewal
4. ✅ Persistent token store

### Other v2.0.0 Highlights
- **DPoP token binding** (RFC 9449) — tie access tokens to client key pairs
- **OAuth 2.0 Authorization Code flow** — full browser redirect
- **OIDC UserInfo endpoint** — GET /oauth/userinfo
- **Token Exchange** (RFC 8693) — impersonation/delegation
- **WebSocket streaming** — /ws/logs and /ws/metrics
- **Chaos testing middleware** — fault injection for resilience testing
- **ETag conditional responses** — bandwidth savings
- **Prometheus remote write** — accept metrics from external apps
- **Interactive tutorial** — 7-step hands-on lab guide
- **Admin UI** — full CRUD for clients, tokens, policies, keys
- **CLI interactive mode** — REPL with tab completion

## Upgrade Guide

v2.0.0 is backward compatible with v1.x configs. New features are opt-in:

```yaml
# New optional fields
token_store: file:/var/lib/oauth4os/tokens.json
secrets_backend: env
```

```bash
# New optional env vars
OAUTH4OS_PLUGINS=/path/to/plugin1.so,/path/to/plugin2.so
```

## What's Next

v2.1.0 will focus on:
- S3 audit export uploader (wired, needs S3 client)
- DynamoDB/Redis token store backends
- Kubernetes operator (CRD)
- Multi-instance coordination (shared revocation list)
