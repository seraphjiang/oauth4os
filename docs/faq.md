# FAQ

20 most common questions about oauth4os.

---

### 1. Does oauth4os modify OpenSearch?

No. The proxy sits in front of OpenSearch and forwards authenticated requests. It sets `X-Proxy-User` and `X-Proxy-Roles` headers that the Security Plugin reads. No patches, no plugins, no config changes to OpenSearch itself.

### 2. Can I use my existing OIDC provider?

Yes. oauth4os works with any OIDC-compliant provider: Keycloak, Auth0, Okta, Dex, Google, Azure AD. Configure the issuer URL and JWKS endpoint in `config.yaml`.

### 3. What happens if the proxy goes down?

Clients can't authenticate through the proxy. If you need direct access as a fallback, keep OpenSearch's native auth (basic auth, API keys) enabled. The proxy is additive — it doesn't disable existing auth methods.

### 4. How do I migrate without downtime?

Deploy oauth4os alongside your existing setup. Migrate clients one at a time from direct access to proxy access. Old and new auth methods work simultaneously. See [docs/migration.md](migration.md).

### 5. What's the performance overhead?

~1-2ms per request. The proxy adds JWT validation, scope mapping, and Cedar evaluation, but these are sub-millisecond operations. The dominant latency is the upstream OpenSearch request. See [docs/performance.md](performance.md).

### 6. Does it work with OpenSearch Serverless (AOSS)?

Yes. The proxy signs upstream requests with AWS SigV4 automatically. Clients use simple Bearer tokens — the proxy translates to SigV4 on the backend. Configure `upstream.sigv4` in config.yaml.

### 7. How are tokens stored?

Tokens are self-contained JWTs — they're not stored on the server. The proxy validates them by checking the signature against cached JWKS keys. Refresh tokens and revocation state are stored in memory (lost on restart) or can be persisted to disk.

### 8. Can I revoke a token?

Yes. `DELETE /oauth/token/{id}` revokes a token immediately. Revoked tokens are rejected on the next request. For refresh tokens, revoking one token revokes the entire token family (all tokens issued from the same original grant).

### 9. What's the difference between scopes and Cedar policies?

Scopes map to OpenSearch backend roles (coarse-grained: "this client can read logs"). Cedar policies add fine-grained rules on top (e.g., "permit reads on logs-*, but forbid access to .opendistro_security"). Cedar is optional — scopes alone work fine for simple setups.

### 10. How does multi-tenancy work?

Each OIDC issuer (provider) is a tenant. Each tenant can have its own scope mappings and Cedar policies. A Keycloak realm and a Dex instance can coexist with completely different authorization rules. Tokens from one tenant can't access another tenant's resources.

### 11. Can browser apps use oauth4os?

Yes. Use the PKCE authorization code flow (RFC 7636). The proxy serves a consent screen where users approve access. See the demo at `/demo` or the [SDK guide](sdk-guide.md#browser-pkce-flow).

### 12. What OAuth RFCs are implemented?

Four: PKCE (RFC 7636), Token Introspection (RFC 7662), Token Exchange (RFC 8693), and Dynamic Client Registration (RFC 7591).

### 13. How do I add a new client?

```bash
curl -X POST https://proxy:8443/oauth/register \
  -H "Content-Type: application/json" \
  -d '{"client_name":"my-agent","scope":"read:logs-*"}'
```

This returns a `client_id` and `client_secret`. See [API reference](api-reference.md#post-oauthregister).

### 14. How do I rotate a client secret?

```bash
curl -X POST https://proxy:8443/oauth/register/{client_id}/rotate
```

The old secret is immediately invalidated. Existing tokens remain valid until they expire.

### 15. Can I use oauth4os without an external OIDC provider?

Yes. The proxy can act as its own token issuer using the built-in `client_credentials` grant. It generates and signs its own JWTs with auto-rotating RSA keys. No external IdP required.

### 16. How do I restrict access by IP?

Configure `ip_filter` in config.yaml:

```yaml
ip_filter:
  clients:
    my-agent:
      allow: ["10.0.0.0/8"]
      deny: ["10.0.1.0/24"]
```

IP filtering is evaluated before JWT validation — blocked IPs never reach the auth layer.

### 17. What log format does the proxy use?

Structured JSON to stdout. Each line is a JSON object with timestamp, level, message, and context fields. Compatible with any log aggregator (Fluentd, CloudWatch, Datadog).

### 18. How do I monitor the proxy?

- `GET /health` — basic health check (for load balancers)
- `GET /health/deep` — checks upstream, JWKS, TLS
- `GET /metrics` — Prometheus-format metrics (9 counters/gauges)
- `GET /admin/analytics` — top clients, scope distribution
- `GET /developer/analytics` — live dashboard (HTML)

### 19. What are the external dependencies?

Two Go modules: `github.com/golang-jwt/jwt/v5` (JWT parsing) and `gopkg.in/yaml.v3` (config). Everything else uses Go's standard library. The binary is ~12MB.

### 20. How do I contribute?

Fork the repo, create a feature branch, run `go test ./...`, and open a PR. See [CONTRIBUTING.md](../CONTRIBUTING.md). All code is Apache 2.0 licensed.
