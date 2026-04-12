# oauth4os — 5-Minute Demo Script

**Target**: YouTube / conference talk / OpenSearch community meeting
**Duration**: 5 minutes
**Format**: Screen recording with voiceover

---

## [0:00–0:20] Hook

> Every observability platform — Grafana, Datadog, Elastic — has OAuth apps and scoped tokens for machine-to-machine access. OpenSearch doesn't. Until now.
>
> oauth4os is an OAuth 2.0 proxy that adds scoped tokens, OIDC federation, Cedar policies, and a token management UI to OpenSearch — with zero changes to your existing cluster.
>
> Let me show you how it works in 5 minutes.

**Screen**: Show the GitHub repo landing page, star count, badges.

---

## [0:20–0:50] Start the Stack

> One command to start everything — the proxy, OpenSearch, and Keycloak as our OIDC provider.

```bash
docker compose up -d
```

**Screen**: Terminal showing containers starting. Cut to `docker ps` showing 3 healthy containers.

> oauth4os sits between your clients and OpenSearch on port 8443. It validates tokens, maps scopes to roles, and forwards requests. Your existing auth — basic auth, OIDC, SAML — still works. oauth4os is additive.

---

## [0:50–1:30] Get a Scoped Token

> Let's get a token scoped to read-only access on log indices.

```bash
curl -s -X POST http://localhost:8443/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=log-reader" \
  -d "client_secret=secret" \
  -d "scope=read:logs-*" | jq .
```

**Screen**: Show the JSON response with `access_token`, `token_type`, `expires_in`, `scope`.

> We got a Bearer token scoped to `read:logs-*`. This token can only read log indices — nothing else. Let's use it.

---

## [1:30–2:10] Query Through the Proxy

> Query OpenSearch through the proxy with our scoped token.

```bash
export TOKEN=<paste token>

# This works — reading logs
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/logs-*/_search \
  -d '{"query":{"match":{"level":"error"}}}' | jq .hits.total

# This fails — no write scope
curl -s -X PUT -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/logs-new/_doc/1 \
  -d '{"message":"test"}'
```

**Screen**: First curl returns results. Second curl returns 403 Forbidden.

> The proxy enforced our scope. Read works, write is blocked. The token only has `read:logs-*`, not `write:logs-*`.

---

## [2:10–2:50] Cedar Policies

> Beyond scopes, oauth4os has Cedar policies for fine-grained control. Here's our config:

```yaml
cedar:
  policies:
    - 'forbid(*, *, .opendistro_security);'
```

> This blocks ALL access to the security index — even admin tokens can't touch it.

```bash
# Even with admin scope, security index is blocked
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:8443/.opendistro_security/_search
```

**Screen**: 403 with `"policy":"forbid-security-index"` in the response.

> Cedar policies are evaluated locally — no network calls, sub-millisecond overhead.

---

## [2:50–3:30] Rate Limiting

> Every client gets rate-limited. Our config gives read scopes 120 RPM and admin 30 RPM.

```bash
# Hammer the proxy
for i in $(seq 1 130); do
  curl -s -o /dev/null -w "%{http_code} " \
    -H "Authorization: Bearer $TOKEN" \
    http://localhost:8443/logs-*/_search
done
```

**Screen**: Show 200s turning to 429s after 120 requests. Show the `Retry-After` header.

> After 120 requests, we get 429 Too Many Requests with a Retry-After header. Protects your cluster from runaway agents.

---

## [3:30–4:10] OSD Plugin + CLI

> oauth4os comes with a token management UI for OpenSearch Dashboards.

**Screen**: Switch to browser showing OSD plugin — token list table with status badges, scope tags.

> List tokens, create new ones, revoke with a confirmation dialog. All from Dashboards.

> Or use the CLI:

```bash
oauth4os login --provider keycloak
oauth4os create-token --scope "read:logs-*" --name ci-pipeline
oauth4os list-tokens
oauth4os revoke tok_abc123
```

**Screen**: Terminal showing CLI output with formatted table.

---

## [4:10–4:40] Monitoring

> The proxy exposes Prometheus metrics at `/metrics`.

```bash
curl http://localhost:8443/metrics
```

**Screen**: Show metrics output — requests_total, auth_success, auth_failed, cedar_denied, rate_limited.

> Pre-built Grafana dashboard included. One command:

```bash
docker compose -f docker-compose.monitoring.yml up
```

**Screen**: Grafana dashboard with request rate, auth failure rate, Cedar denials.

---

## [4:40–5:00] Wrap Up

> oauth4os adds what OpenSearch is missing — scoped tokens, OIDC federation, Cedar policies, rate limiting, and a management UI. Zero changes to your cluster.
>
> It's open source, Apache 2.0, and built as a response to RFC 491 in the OpenSearch project.
>
> Star us on GitHub, try the quickstart, and let us know what you think.

**Screen**: GitHub repo with star button highlighted. Link to RFC #491.

> Links in the description. Thanks for watching.

---

## Production Notes

- **Resolution**: 1920×1080, dark terminal theme
- **Terminal font**: JetBrains Mono or SF Mono, 16px
- **Browser**: Firefox/Chrome with dark mode
- **jq**: Use for pretty-printing JSON responses
- **Cuts**: Hard cuts between sections, no transitions
- **Music**: None — voiceover only
- **Captions**: Auto-generate, review for accuracy
