# Migration Guide — Adding oauth4os to Your OpenSearch Cluster

This guide walks you through adding oauth4os alongside your existing OpenSearch Security Plugin setup. oauth4os is additive — your existing auth continues to work unchanged.

## Prerequisites

- OpenSearch 2.x or 3.x cluster with Security Plugin enabled
- An OIDC provider (Keycloak, Auth0, Okta, Dex, or Google)
- Docker (for proxy) or Go 1.22+ (for building from source)

## Step 1: Assess Your Current Setup

Check what auth methods you're currently using:

```bash
# Check Security Plugin config
curl -k -u admin:admin https://opensearch:9200/_plugins/_security/api/securityconfig
```

Common setups:
- **Basic auth only** → oauth4os adds token-based access
- **OIDC for humans** → oauth4os adds machine-to-machine access
- **SAML for SSO** → oauth4os adds API access alongside SSO

oauth4os works with all of these. No changes needed to your existing config.

## Step 2: Create Backend Roles

Create OpenSearch roles that oauth4os will map scopes to:

```bash
# Create a read-only role for logs
curl -k -u admin:admin -X PUT https://opensearch:9200/_plugins/_security/api/roles/logs_read_access \
  -H 'Content-Type: application/json' \
  -d '{
    "cluster_permissions": ["cluster_composite_ops_ro"],
    "index_permissions": [{
      "index_patterns": ["logs-*"],
      "allowed_actions": ["read", "search"]
    }]
  }'

# Create a dashboards write role
curl -k -u admin:admin -X PUT https://opensearch:9200/_plugins/_security/api/roles/dashboard_write_access \
  -H 'Content-Type: application/json' \
  -d '{
    "cluster_permissions": ["cluster_composite_ops"],
    "index_permissions": [{
      "index_patterns": [".kibana*", ".opensearch_dashboards*"],
      "allowed_actions": ["crud"]
    }]
  }'
```

## Step 3: Configure oauth4os

Create `config.yaml`:

```yaml
upstream:
  engine: https://your-opensearch:9200
  dashboards: https://your-dashboards:5601

providers:
  - name: your-provider
    issuer: https://your-oidc-provider.com/realms/opensearch
    jwks_uri: auto  # auto-discovers from .well-known/openid-configuration

scope_mapping:
  "read:logs-*":
    backend_roles: [logs_read_access]
  "write:dashboards":
    backend_roles: [dashboard_write_access]
  "admin":
    backend_roles: [all_access]

rate_limit:
  default_rpm: 60

listen: :8443

tls:
  enabled: true
  cert_file: /etc/oauth4os/tls.crt
  key_file: /etc/oauth4os/tls.key
```

## Step 4: Deploy the Proxy

### Option A: Docker

```bash
docker run -d \
  --name oauth4os \
  -p 8443:8443 \
  -v $(pwd)/config.yaml:/etc/oauth4os/config.yaml \
  ghcr.io/seraphjiang/oauth4os:latest \
  --config /etc/oauth4os/config.yaml
```

### Option B: Helm

```bash
helm install oauth4os deploy/helm/oauth4os/ \
  --set config.upstream.engine=https://opensearch:9200 \
  --set config.upstream.dashboards=https://dashboards:5601
```

### Option C: Binary

```bash
# Download from GitHub Releases
oauth4os-proxy --config config.yaml
```

## Step 5: Register Clients

Register OAuth clients in your OIDC provider. For Keycloak:

1. Create a new client with `client_credentials` grant type
2. Set client ID and secret
3. Configure allowed scopes (`read:logs-*`, `write:dashboards`, etc.)

For testing without an OIDC provider, oauth4os has a built-in token manager:

```bash
curl -X POST http://localhost:8443/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=my-agent" \
  -d "client_secret=secret" \
  -d "scope=read:logs-*"
```

## Step 6: Update Client Applications

Point your machine clients at the proxy instead of OpenSearch directly:

```diff
- OPENSEARCH_URL=https://opensearch:9200
+ OPENSEARCH_URL=https://oauth4os-proxy:8443
```

Add token acquisition to your client:

```python
# Before: basic auth
# client = OpenSearch(hosts=[{'host': 'opensearch', 'port': 9200}], http_auth=('admin', 'admin'))

# After: oauth4os token
import requests
token_resp = requests.post('https://oauth4os-proxy:8443/oauth/token', data={
    'grant_type': 'client_credentials',
    'client_id': 'my-agent',
    'client_secret': 'secret',
    'scope': 'read:logs-*'
})
token = token_resp.json()['access_token']

client = OpenSearch(
    hosts=[{'host': 'oauth4os-proxy', 'port': 8443}],
    headers={'Authorization': f'Bearer {token}'}
)
```

## Step 7: Verify

```bash
# Health check
curl https://oauth4os-proxy:8443/health

# Get token and query
TOKEN=$(curl -s -X POST https://oauth4os-proxy:8443/oauth/token \
  -d "grant_type=client_credentials&client_id=my-agent&client_secret=secret&scope=read:logs-*" \
  | jq -r .access_token)

curl -H "Authorization: Bearer $TOKEN" \
  https://oauth4os-proxy:8443/logs-*/_search \
  -d '{"query":{"match_all":{}}}'

# Verify scope enforcement — this should fail (no write scope)
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  https://oauth4os-proxy:8443/logs-new/_doc/1 \
  -d '{"message":"test"}'
# Expected: 403 Forbidden
```

## Step 8: Install OSD Plugin (Optional)

For token management UI in OpenSearch Dashboards:

```bash
cd plugins/oauth4os-dashboards
# Follow OSD plugin installation for your version
```

## Rollback

oauth4os is a proxy — removing it is as simple as pointing clients back to OpenSearch directly. No data migration, no config changes to OpenSearch.

```diff
- OPENSEARCH_URL=https://oauth4os-proxy:8443
+ OPENSEARCH_URL=https://opensearch:9200
```

## Coexistence Diagram

```
                    ┌─────────────────────────────────┐
                    │         Your Cluster             │
                    │                                  │
Humans ──── OSD ───▶│  Security Plugin (OIDC/SAML)    │
                    │         ↓                        │
                    │    OpenSearch Engine              │
                    │         ↑                        │
Machines ── oauth4os proxy ──▶│  (same cluster, same data)  │
                    │                                  │
                    └─────────────────────────────────┘
```

Both auth paths coexist. Humans use OSD + OIDC/SAML. Machines use oauth4os + scoped tokens. Same cluster, same indices, same data.
