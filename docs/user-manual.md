# oauth4os User Manual

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Configuration](#configuration)
4. [OIDC Provider Setup](#oidc-provider-setup)
5. [Token Management](#token-management)
6. [Scope & Role Mapping](#scope--role-mapping)
7. [Cedar Policies](#cedar-policies)
8. [CLI Reference](#cli-reference)
9. [API Reference](#api-reference)
10. [AI Agent Integration](#ai-agent-integration)
11. [Deployment](#deployment)
12. [Monitoring](#monitoring)
13. [Troubleshooting](#troubleshooting)

---

## Overview

oauth4os is an OAuth 2.0 proxy for OpenSearch. It sits between clients and OpenSearch, validating JWT tokens, mapping OAuth scopes to OpenSearch security roles, and forwarding authenticated requests.

```
Client → oauth4os (:8443) → OpenSearch Engine (:9200)
              ↕                      ↕
         OIDC Provider          Dashboards (:5601)
```

**Key concepts:**
- **Proxy** — validates tokens, maps scopes, forwards requests
- **Scope** — what a token is allowed to do (e.g., `read:logs-*`)
- **Role mapping** — scopes map to OpenSearch backend roles
- **Cedar policy** — optional fine-grained access control layer
- **Passthrough** — requests without Bearer tokens go directly to OpenSearch (existing auth applies)

---

## Installation

### Docker (recommended)

```bash
docker pull ghcr.io/seraphjiang/oauth4os:latest
docker run -p 8443:8443 -v ./config.yaml:/etc/oauth4os/config.yaml ghcr.io/seraphjiang/oauth4os
```

### Docker Compose (with OpenSearch + Keycloak)

```bash
git clone https://github.com/seraphjiang/oauth4os.git
cd oauth4os
docker compose -f docker-compose.demo.yml up -d
```

This starts OpenSearch, Dashboards, Keycloak (OIDC provider), and the proxy — all pre-configured.

### Helm (Kubernetes)

```bash
helm install oauth4os deploy/helm/oauth4os \
  --set config.upstream.engine=https://opensearch:9200 \
  --set config.upstream.dashboards=https://dashboards:5601 \
  --set config.providers[0].issuer=https://keycloak.example.com/realms/opensearch
```

### From Source

```bash
go install github.com/seraphjiang/oauth4os/cmd/proxy@latest
oauth4os -config config.yaml
```

---

## Configuration

Configuration is a YAML file passed via `-config` flag.

### Full Reference

```yaml
# Upstream OpenSearch endpoints
upstream:
  engine: https://opensearch:9200        # Required
  dashboards: https://dashboards:5601    # Optional — enables Dashboards API proxying

# OIDC providers (one or more)
providers:
  - name: keycloak                       # Display name
    issuer: https://keycloak.example.com/realms/opensearch  # OIDC issuer URL
    jwks_uri: auto                       # "auto" = discover from .well-known/openid-configuration

  - name: auth0
    issuer: https://mycompany.auth0.com
    jwks_uri: auto

# Scope-to-role mapping
scope_mapping:
  "read:logs-*":                         # OAuth scope string
    backend_user: logs-reader            # OpenSearch internal user (optional)
    backend_roles:                       # OpenSearch backend roles
      - logs_read_access

  "write:logs-*":
    backend_roles: [logs_write_access]

  "admin":
    backend_roles: [all_access]

# Cedar policy engine (optional)
cedar:
  enabled: false                         # Set true to enable
  policies:                              # Cedar policy strings
    - 'permit(*, *, *);'
    - 'forbid(*, *, .opendistro_security);'

# Network
listen: :8443                            # Proxy listen address

# TLS (optional)
tls:
  cert: /path/to/cert.pem
  key: /path/to/key.pem
  insecure_skip_verify: false            # Skip upstream TLS verification (dev only)
```

### Environment Variable Overrides

| Variable | Overrides | Example |
|----------|-----------|---------|
| `OAUTH4OS_LISTEN` | `listen` | `:8443` |
| `OAUTH4OS_ENGINE_URL` | `upstream.engine` | `https://opensearch:9200` |
| `OAUTH4OS_DASHBOARDS_URL` | `upstream.dashboards` | `https://dashboards:5601` |

---

## OIDC Provider Setup

oauth4os works with any OIDC-compliant provider. Tested with:

### Keycloak

1. Create a realm (e.g., `opensearch`)
2. Create client scopes: `read:logs-*`, `write:logs-*`, `admin`
3. Create clients with `Service accounts roles` enabled
4. Assign scopes to clients

Or use the bundled realm: `deploy/keycloak/realm-export.json` — pre-configured with 4 clients.

### Auth0

1. Create an API with identifier `https://opensearch.example.com`
2. Define permissions: `read:logs-*`, `write:logs-*`, `admin`
3. Create Machine-to-Machine applications
4. Grant permissions to each application

Config:
```yaml
providers:
  - name: auth0
    issuer: https://your-tenant.auth0.com/
    jwks_uri: auto
```

### Dex

```yaml
providers:
  - name: dex
    issuer: https://dex.example.com
    jwks_uri: https://dex.example.com/keys
```

### Multiple Providers

oauth4os supports multiple providers simultaneously. The proxy matches tokens to providers by the `iss` (issuer) claim.

---

## Token Management

### Get a Token (Client Credentials)

```bash
curl -X POST https://keycloak.example.com/realms/opensearch/protocol/openid-connect/token \
  -d "grant_type=client_credentials" \
  -d "client_id=my-agent" \
  -d "client_secret=my-secret" \
  -d "scope=read:logs-*"
```

Response:
```json
{
  "access_token": "eyJhbGciOi...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read:logs-*"
}
```

### Use a Token

```bash
curl -H "Authorization: Bearer eyJhbGciOi..." \
  https://proxy:8443/logs-*/_search \
  -d '{"query":{"match_all":{}}}'
```

### Proxy Token API

The proxy also has its own token management endpoints:

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/oauth/token` | Issue a proxy-managed token |
| `GET` | `/oauth/tokens` | List active tokens |
| `GET` | `/oauth/token/{id}` | Get token details |
| `DELETE` | `/oauth/token/{id}` | Revoke a token |

### Token Lifecycle

```
Issue → Active → [Expire | Revoke] → Inactive
```

- Tokens expire based on the OIDC provider's `expires_in`
- Revoked tokens are immediately rejected
- The proxy caches JWKS keys for 1 hour

---

## Scope & Role Mapping

Scopes in the JWT `scope` claim are mapped to OpenSearch backend roles.

### How It Works

1. Client requests token with `scope=read:logs-*`
2. OIDC provider issues JWT with `"scope": "read:logs-*"`
3. Proxy extracts scopes from JWT
4. Proxy looks up `scope_mapping` in config
5. Proxy injects mapped `backend_roles` into the request
6. OpenSearch FGAC enforces the role's permissions

### Example Mappings

```yaml
scope_mapping:
  # AI agent — read-only access to log indices
  "read:logs-*":
    backend_roles: [logs_read_access]

  # Fluent Bit — write-only to its indices
  "write:logs-*":
    backend_roles: [logs_write_access]

  # CI/CD — manage dashboards
  "write:dashboards":
    backend_roles: [dashboard_write_access]

  # Admin — full access
  "admin":
    backend_roles: [all_access]
```

### Multiple Scopes

A token can have multiple scopes. All mapped roles are combined:

```
Token scope: "read:logs-* read:metrics-*"
→ Roles: [logs_read_access, metrics_read_access]
```

---

## Cedar Policies

Cedar provides fine-grained access control beyond scope mapping.

### Enable

```yaml
cedar:
  enabled: true
  policies:
    - 'permit(*, GET, logs-*);'
    - 'forbid(*, *, .opendistro_security);'
    - 'forbid(*, DELETE, *) unless { principal.role == "admin" };'
```

### Policy Syntax

```
permit|forbid(principal, action, resource) [when|unless { conditions }];
```

- `*` — matches anything
- `principal` — the client_id or role from the token
- `action` — HTTP method (GET, POST, PUT, DELETE)
- `resource` — the OpenSearch index from the URL path

### Evaluation Order

1. All `forbid` rules are checked first (forbid-overrides)
2. If any `forbid` matches → **403 Forbidden**
3. Then `permit` rules are checked
4. If any `permit` matches → **allowed**
5. No match → **403 Forbidden** (default deny)

### Examples

```
# Allow everyone to search logs
permit(*, GET, logs-*);

# Block access to security index
forbid(*, *, .opendistro_security);

# Only admins can delete
forbid(*, DELETE, *) unless { principal.role == "admin" };

# Specific client can write to specific index
permit(*, POST, metrics-*) when { principal.scope contains "write:metrics" };
```

---

## CLI Reference

Install: `go install github.com/seraphjiang/oauth4os/cmd/cli@latest`

### Commands

| Command | Description |
|---------|-------------|
| `oauth4os login <client_id> <secret> [scope]` | Authenticate and cache token |
| `oauth4os logout` | Clear cached credentials |
| `oauth4os create-token <client_id> [scope]` | Issue a new token |
| `oauth4os revoke-token <token_id>` | Revoke a token |
| `oauth4os list-tokens` | List active tokens |
| `oauth4os inspect-token <token_id>` | Show token details |
| `oauth4os status` | Show current auth state |
| `oauth4os config` | Show configuration |

### Config File

`~/.oauth4os.yaml` — created on first `login`:

```yaml
server: http://localhost:8443
token_url: http://localhost:8080/realms/opensearch/protocol/openid-connect/token
client_id: log-reader
token: eyJhbGciOi...
token_expiry: 2025-01-15T11:30:00Z
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `OAUTH4OS_SERVER` | Proxy URL |
| `OAUTH4OS_TOKEN` | Bearer token (overrides cached) |

---

## API Reference

### Proxy Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | None | Health check + version |
| `GET` | `/metrics` | None | Prometheus metrics |
| `POST` | `/oauth/token` | Client credentials | Issue token |
| `GET` | `/oauth/tokens` | Bearer | List tokens |
| `GET` | `/oauth/token/{id}` | Bearer | Token details |
| `DELETE` | `/oauth/token/{id}` | Bearer | Revoke token |
| `*` | `/api/*` | Bearer or passthrough | Proxied to Dashboards |
| `*` | `/*` | Bearer or passthrough | Proxied to Engine |

### Prometheus Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `oauth4os_requests_total` | Counter | Total proxied requests |
| `oauth4os_requests_active` | Gauge | Currently active requests |
| `oauth4os_requests_failed` | Counter | Failed requests (4xx/5xx) |
| `oauth4os_auth_success` | Counter | Successful token validations |
| `oauth4os_auth_failed` | Counter | Failed token validations |
| `oauth4os_cedar_denied` | Counter | Requests denied by Cedar |
| `oauth4os_upstream_errors` | Counter | Upstream connection errors |
| `oauth4os_uptime_seconds` | Gauge | Proxy uptime |

---

## AI Agent Integration

### MCP Server (Claude Desktop, Cursor)

```json
{
  "mcpServers": {
    "opensearch": {
      "command": "python",
      "args": ["path/to/oauth4os/examples/mcp-server/server.py"],
      "env": {
        "OAUTH4OS_URL": "https://proxy:8443",
        "OAUTH4OS_TOKEN_URL": "https://keycloak/realms/opensearch/protocol/openid-connect/token",
        "OAUTH4OS_CLIENT_ID": "log-reader",
        "OAUTH4OS_CLIENT_SECRET": "log-reader-secret",
        "OAUTH4OS_SCOPE": "read:logs-*"
      }
    }
  }
}
```

Available tools: `search_logs`, `aggregate`, `get_indices`, `get_mappings`, `create_index`, `delete_docs`, `get_cluster_health`

### LangChain / LlamaIndex

```python
from opensearchpy import OpenSearch

client = OpenSearch(
    hosts=[{"host": "proxy", "port": 8443}],
    headers={"Authorization": f"Bearer {token}"},
    use_ssl=True
)
```

### Any HTTP Client

```bash
curl -H "Authorization: Bearer $TOKEN" https://proxy:8443/logs-*/_search
```

---

## Deployment

### Docker

```bash
docker run -p 8443:8443 \
  -v ./config.yaml:/etc/oauth4os/config.yaml \
  ghcr.io/seraphjiang/oauth4os:latest
```

### Kubernetes (Helm)

```bash
helm install oauth4os deploy/helm/oauth4os -f values.yaml
```

### AWS (CDK)

```bash
cd deploy/cdk
cdk deploy OAuth4OS \
  -c domain_base=example.com \
  -c hosted_zone_id=Z123 \
  -c certificate_arn=arn:aws:acm:...
```

### Sidecar Pattern

Run oauth4os as a sidecar alongside OpenSearch:

```yaml
# K8s pod spec
containers:
  - name: opensearch
    image: opensearchproject/opensearch:latest
    ports: [{containerPort: 9200}]
  - name: oauth-proxy
    image: ghcr.io/seraphjiang/oauth4os:latest
    ports: [{containerPort: 8443}]
    volumeMounts:
      - name: config
        mountPath: /etc/oauth4os
```

---

## Monitoring

### Health Check

```bash
curl http://proxy:8443/health
# {"status":"ok","version":"0.2.0","uptime_seconds":3600}
```

### Prometheus

Scrape `/metrics` on the proxy port. Example Grafana dashboard queries:

```promql
# Request rate
rate(oauth4os_requests_total[5m])

# Auth failure rate
rate(oauth4os_auth_failed[5m]) / rate(oauth4os_requests_total[5m])

# Cedar deny rate
rate(oauth4os_cedar_denied[5m])
```

### Audit Log

Every authenticated request is logged to stdout:

```
[2025-01-15T10:30:00Z] client=log-reader scopes=[read:logs-*] GET /logs-*/_search
[2025-01-15T10:30:01Z] client=ci-pipeline scopes=[write:dashboards] POST /api/saved_objects/_import
```

---

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `401 invalid_token` | JWT expired or malformed | Get a fresh token from OIDC provider |
| `401 unknown issuer` | Token issuer not in config | Add provider to `providers` list |
| `403 insufficient_scope` | Token scopes don't map to any roles | Check `scope_mapping` in config |
| `403 cedar_denied` | Cedar policy blocked the request | Review Cedar policies |
| `502 Bad Gateway` | Can't reach upstream OpenSearch | Check `upstream.engine` URL |
| JWKS fetch timeout | OIDC provider unreachable | Check network, verify `jwks_uri` |
| Token not refreshing | CLI cache stale | Run `oauth4os logout && oauth4os login` |

### Debug Mode

```bash
OAUTH4OS_DEBUG=true oauth4os -config config.yaml
```

Logs all JWT claims, scope mappings, and Cedar evaluations.
