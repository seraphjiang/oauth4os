# oauth4os v0.2.0 — Developer Portal, Live Demo, and OpenSearch Serverless

We shipped v0.2.0 of [oauth4os](https://github.com/seraphjiang/oauth4os), the OAuth 2.0 proxy for OpenSearch. This release adds a developer portal, two interactive demo experiences, and native AWS OpenSearch Serverless (AOSS) support.

**Try it now**: https://f5cmk2hxwx.us-west-2.awsapprunner.com

No setup, no sign-up. The live demo is backed by real OpenSearch Serverless with 500+ searchable log entries.

---

## What's new in v0.2.0

### Developer Portal

The proxy now serves a built-in developer portal at `/developer/docs` with OpenAPI documentation and at `/developer/analytics` with a live analytics dashboard showing top clients, scope distribution, and request timelines.

Operators get visibility into who's using the proxy and how — without deploying a separate monitoring stack.

### Demo Web App — PKCE in Action

Visit [/demo](https://f5cmk2hxwx.us-west-2.awsapprunner.com/demo) to see the full OAuth browser flow:

1. Click "Login with oauth4os"
2. The app generates a PKCE challenge and redirects to `/oauth/authorize`
3. The consent screen shows the requesting app and its scopes
4. Approve → the app receives an authorization code → exchanges it for a token
5. The dashboard loads with real log data from OpenSearch

The demo also includes a scope enforcement section — try searching with `read:logs-*` (allowed) versus writing data (denied). This is the experience we want every OpenSearch OAuth integration to have.

### CLI Installer

One command to get started from the terminal:

```bash
curl -sL https://f5cmk2hxwx.us-west-2.awsapprunner.com/install.sh | bash
```

This installs `oauth4os-demo`, a shell wrapper that handles PKCE login, token caching, and search:

```bash
oauth4os-demo login          # Opens browser → consent → token cached
oauth4os-demo search 'level:ERROR'   # Search through the proxy
oauth4os-demo services       # List indexed services
oauth4os-demo health         # Check proxy status
```

### OpenSearch Serverless (AOSS) Integration

The proxy now signs upstream requests with AWS SigV4 when targeting AOSS collections. Configure it in `config.yaml`:

```yaml
upstream:
  engine: https://abc123.us-west-2.aoss.amazonaws.com
  sigv4:
    enabled: true
    region: us-west-2
    service: aoss
```

The proxy handles credential refresh, request signing, and header cleanup transparently. Your clients still use OAuth tokens — the proxy translates to SigV4 on the backend.

### Consent Screen

PKCE authorization now shows a proper consent screen at `/oauth/consent`:

- App name and icon
- Each requested scope with a human-readable description
- Write/admin scopes trigger a warning banner
- Approve or deny with one click

This is what users see when an OAuth app requests access to their OpenSearch data.

### Client Management

Full CRUD for OAuth clients via the REST API:

```bash
# Register a new client
curl -X POST https://f5cmk2hxwx.us-west-2.awsapprunner.com/oauth/register \
  -H "Content-Type: application/json" \
  -d '{"client_name": "my-agent", "scope": "read:logs-*"}'

# Rotate a client secret
curl -X POST https://f5cmk2hxwx.us-west-2.awsapprunner.com/oauth/register/<client_id>/rotate
```

Clients persist across restarts with atomic writes and automatic backup on rotation.

---

## Try it yourself

**Get a token and search:**

```bash
TOKEN=$(curl -sf -X POST https://f5cmk2hxwx.us-west-2.awsapprunner.com/oauth/token \
  -d "grant_type=client_credentials&client_id=demo-agent&client_secret=demo-secret&scope=read:logs-*" \
  | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

# Search for errors
curl -sf -H "Authorization: Bearer $TOKEN" \
  "https://f5cmk2hxwx.us-west-2.awsapprunner.com/logs-demo/_search" \
  -H "Content-Type: application/json" \
  -d '{"query":{"term":{"level":"ERROR"}},"size":5}' | python3 -m json.tool
```

**Check the OIDC discovery:**

```bash
curl -sf https://f5cmk2hxwx.us-west-2.awsapprunner.com/.well-known/openid-configuration | python3 -m json.tool
```

**Inspect the JWKS:**

```bash
curl -sf https://f5cmk2hxwx.us-west-2.awsapprunner.com/.well-known/jwks.json | python3 -m json.tool
```

---

## By the numbers

| Metric | v0.1.0 | v0.2.0 |
|--------|--------|--------|
| Commits | 170 | 218 |
| Go source | 4,500 lines | 6,500 lines |
| Test functions | 356 | 371 |
| Internal packages | 20 | 25 |
| API endpoints | 12 | 20 |

New packages in v0.2.0: `demo`, `sigv4`, `backup`, `webhook`, `registration`.

---

## What's next

We're working on v0.3.0:

- **GitHub Actions CI** — automated build, test, and release pipeline
- **OAuth Playground** — interactive page that walks through each PKCE step visually
- **KQL support** — parse Kusto Query Language in the CLI (`oauth4os-demo search 'service:payment AND level:ERROR'`)
- **Stress testing** — p50/p95/p99 latency numbers under 1000 concurrent requests

## Get involved

This project implements the OAuth proxy approach proposed in [opensearch-project/.github#491](https://github.com/opensearch-project/.github/issues/491). We'd love feedback from the OpenSearch community:

- **Source**: https://github.com/seraphjiang/oauth4os
- **Live demo**: https://f5cmk2hxwx.us-west-2.awsapprunner.com
- **RFC discussion**: https://github.com/opensearch-project/.github/issues/491

All code is Apache 2.0 licensed. PRs welcome.
