# Migration Guide

Step-by-step guides for migrating to oauth4os from common OpenSearch auth setups. oauth4os is additive — your existing auth continues to work alongside the proxy.

---

## Migration from Basic Auth

The most common setup: clients authenticate with `admin:admin` or per-user credentials.

### Before

```bash
# Every client has the admin password
curl -k -u admin:admin https://opensearch:9200/logs-demo/_search \
  -d '{"query":{"match_all":{}}}'
```

Problems:
- Shared credentials — can't revoke one client without changing everyone's password
- No audit trail per client
- No scope restrictions — every client has full admin access
- Credentials in plaintext in scripts, CI configs, environment variables

### After

```bash
# Each client gets its own scoped token
TOKEN=$(curl -sf -X POST https://proxy:8443/oauth/token \
  -d "grant_type=client_credentials&client_id=ci-agent&client_secret=ci-secret&scope=read:logs-*")

curl -H "Authorization: Bearer $TOKEN" https://proxy:8443/logs-demo/_search \
  -d '{"query":{"match_all":{}}}'
```

### Migration Steps

**Step 1**: Deploy oauth4os in front of OpenSearch (basic auth still works directly):

```
Clients → oauth4os (:8443) → OpenSearch (:9200)
              │                     ↑
              │                     │
              └── basic auth still works directly
```

```yaml
# config.yaml
upstream:
  engine: https://opensearch:9200

providers:
  - name: self
    issuer: https://proxy:8443
    jwks_uri: auto

scope_mapping:
  "read:logs-*":
    backend_roles: [logs_read_access]
  "write:logs-*":
    backend_roles: [logs_write_access]
  "admin":
    backend_roles: [all_access]

listen: :8443
tls:
  insecure_skip_verify: true  # if OpenSearch uses self-signed certs
```

**Step 2**: Create OpenSearch backend roles that match your scope mapping:

```bash
curl -k -u admin:admin -X PUT \
  https://opensearch:9200/_plugins/_security/api/roles/logs_read_access \
  -H "Content-Type: application/json" \
  -d '{
    "cluster_permissions": ["cluster_composite_ops_ro"],
    "index_permissions": [{
      "index_patterns": ["logs-*"],
      "allowed_actions": ["read", "search"]
    }]
  }'
```

**Step 3**: Register clients and issue tokens:

```bash
# Register a client
curl -X POST https://proxy:8443/oauth/register \
  -H "Content-Type: application/json" \
  -d '{"client_name":"ci-agent","scope":"read:logs-*"}'
# Returns: {"client_id":"abc123","client_secret":"sec-xyz..."}

# Test with token
TOKEN=$(curl -sf -X POST https://proxy:8443/oauth/token \
  -d "grant_type=client_credentials&client_id=abc123&client_secret=sec-xyz&scope=read:logs-*" \
  | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

curl -H "Authorization: Bearer $TOKEN" https://proxy:8443/logs-demo/_search
```

**Step 4**: Migrate clients one at a time. Update each script/service to use tokens instead of basic auth. Old clients continue working via direct OpenSearch access.

**Step 5**: Once all clients are migrated, restrict direct OpenSearch access to the proxy's IP only (firewall/security group).

---

## Migration from API Keys (OpenSearch 3.7+)

OpenSearch 3.7 introduced API keys. oauth4os adds scoped tokens, rate limiting, Cedar policies, and audit logging on top.

### Before

```bash
# Create an API key (OpenSearch native)
API_KEY=$(curl -k -u admin:admin -X POST \
  https://opensearch:9200/_plugins/_security/api/apitokens \
  -d '{"name":"my-key","cluster_permissions":["cluster_all"]}' \
  | jq -r '.token')

curl -H "Authorization: ApiKey $API_KEY" \
  https://opensearch:9200/logs-demo/_search
```

Problems:
- No rate limiting per key
- No fine-grained Cedar policies
- No token exchange for federation
- No PKCE flow for browser apps
- Limited audit trail

### After

```bash
TOKEN=$(curl -sf -X POST https://proxy:8443/oauth/token \
  -d "grant_type=client_credentials&client_id=my-agent&client_secret=secret&scope=read:logs-*")

curl -H "Authorization: Bearer $TOKEN" https://proxy:8443/logs-demo/_search
```

### Migration Steps

**Step 1**: Deploy oauth4os. API keys continue to work for direct OpenSearch access.

**Step 2**: Map your API key permissions to oauth4os scopes:

| API Key Permission | oauth4os Scope | Backend Role |
|-------------------|----------------|--------------|
| `cluster_all` | `admin` | `all_access` |
| `index read` on `logs-*` | `read:logs-*` | `logs_read_access` |
| `index write` on `logs-*` | `write:logs-*` | `logs_write_access` |

**Step 3**: Register equivalent clients in oauth4os:

```bash
curl -X POST https://proxy:8443/oauth/register \
  -H "Content-Type: application/json" \
  -d '{"client_name":"my-agent","scope":"read:logs-*"}'
```

**Step 4**: Add Cedar policies for fine-grained control (optional):

```yaml
cedar_policies:
  - "permit(*, GET, logs-*)"
  - "permit(*, POST, logs-*/_search)"
  - "forbid(*, *, .opendistro_*)"
  - "forbid(*, DELETE, *)"
```

**Step 5**: Add rate limiting (optional):

```yaml
rate_limits:
  "read:logs-*": 600
  "admin": 60
```

**Step 6**: Migrate clients from API keys to OAuth tokens. Retire API keys once migrated.

---

## Migration from nginx proxy_pass

Common pattern: nginx sits in front of OpenSearch, handling TLS termination and basic access control.

### Before

```nginx
# nginx.conf
upstream opensearch {
    server opensearch:9200;
}

server {
    listen 443 ssl;
    ssl_certificate /etc/nginx/tls.crt;
    ssl_certificate_key /etc/nginx/tls.key;

    location / {
        proxy_pass https://opensearch;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;

        # Basic auth or IP allowlist
        allow 10.0.0.0/8;
        deny all;
    }
}
```

Problems:
- No per-client authentication (IP-based only)
- No token management or revocation
- No audit trail per client
- No scope-based access control
- Manual nginx config changes for each new client

### After

Replace nginx with oauth4os (or put oauth4os behind nginx):

```yaml
# config.yaml
upstream:
  engine: https://opensearch:9200

providers:
  - name: self
    issuer: https://proxy:8443
    jwks_uri: auto

scope_mapping:
  "read:logs-*":
    backend_roles: [logs_read_access]
  "admin":
    backend_roles: [all_access]

listen: :8443

ip_filter:
  global:
    allow: ["10.0.0.0/8"]

rate_limits:
  "read:logs-*": 600
  "admin": 60
```

### Migration Steps

**Option A: Replace nginx entirely**

```
Before:  Clients → nginx (:443) → OpenSearch (:9200)
After:   Clients → oauth4os (:8443) → OpenSearch (:9200)
```

1. Deploy oauth4os with the same TLS certs:
   ```yaml
   listen: :443
   tls:
     enabled: true
     cert_file: /etc/oauth4os/tls.crt
     key_file: /etc/oauth4os/tls.key
   ```
2. Migrate IP allowlists to oauth4os `ip_filter` config
3. Register clients and issue tokens
4. Update DNS/load balancer to point to oauth4os
5. Decommission nginx

**Option B: Keep nginx for TLS, add oauth4os behind it**

```
Clients → nginx (:443) → oauth4os (:8443) → OpenSearch (:9200)
```

```nginx
# Updated nginx.conf
server {
    listen 443 ssl;
    ssl_certificate /etc/nginx/tls.crt;
    ssl_certificate_key /etc/nginx/tls.key;

    location / {
        proxy_pass http://oauth4os:8443;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

This preserves nginx for TLS termination and adds oauth4os for auth.

---

## Migration from OIDC (Dashboards only)

Many clusters use OIDC for Dashboards login but have no token-based access for APIs.

### Before

```yaml
# opensearch_dashboards.yml
opensearch_security.auth.type: openid
opensearch_security.openid.connect_url: https://keycloak.example.com/realms/opensearch/.well-known/openid-configuration
opensearch_security.openid.client_id: opensearch-dashboards
opensearch_security.openid.client_secret: secret
```

Humans log into Dashboards via OIDC. Scripts use basic auth or have no API access.

### After

Same OIDC provider, now also serving machine-to-machine tokens:

```yaml
# oauth4os config.yaml
upstream:
  engine: https://opensearch:9200
  dashboards: https://opensearch-dashboards:5601

providers:
  - name: keycloak
    issuer: https://keycloak.example.com/realms/opensearch
    jwks_uri: auto

scope_mapping:
  "read:logs-*":
    backend_roles: [logs_read_access]
  "admin":
    backend_roles: [all_access]
```

### Migration Steps

1. Deploy oauth4os pointing to the same OIDC provider
2. Create a Keycloak client for oauth4os (client credentials grant):
   - Client ID: `oauth4os-proxy`
   - Access Type: confidential
   - Service Accounts Enabled: yes
3. Register machine clients in oauth4os
4. Dashboards OIDC login continues unchanged
5. API clients now use oauth4os tokens

```
Humans  → Dashboards (:5601) → OIDC login (unchanged)
Scripts → oauth4os (:8443)   → Token-based access (new)
Both    → Same OpenSearch cluster, same roles
```

---

## Migration from AWS OpenSearch with IAM

If you're using IAM auth directly with OpenSearch Service or Serverless.

### Before

```bash
# Every client needs AWS credentials + SigV4 signing
aws opensearch-serverless batch-get-collection --ids abc123

# Or using a SigV4-aware HTTP client
curl --aws-sigv4 "aws:amz:us-west-2:aoss" \
  --user "$AWS_ACCESS_KEY_ID:$AWS_SECRET_ACCESS_KEY" \
  https://abc123.us-west-2.aoss.amazonaws.com/logs-demo/_search
```

Problems:
- Every client needs AWS IAM credentials
- No OAuth scopes — IAM policies are coarse-grained
- No token revocation (IAM keys are long-lived)
- No Cedar policies
- Complex SigV4 signing in every client

### After

```bash
# Clients use simple Bearer tokens — proxy handles SigV4
TOKEN=$(curl -sf -X POST https://proxy:8443/oauth/token \
  -d "grant_type=client_credentials&client_id=agent&client_secret=secret&scope=read:logs-*")

curl -H "Authorization: Bearer $TOKEN" https://proxy:8443/logs-demo/_search
```

### Migration Steps

1. Deploy oauth4os with SigV4 signing:
   ```yaml
   upstream:
     engine: https://abc123.us-west-2.aoss.amazonaws.com
     sigv4:
       region: us-west-2
       service: aoss
   ```
2. Give the proxy's IAM role AOSS access (see [deployment guide](deployment.md))
3. Register clients in oauth4os — they no longer need AWS credentials
4. Add Cedar policies for fine-grained index-level control
5. Clients use simple Bearer tokens; proxy translates to SigV4

---

## Rollback Plan

If you need to roll back at any point:

1. **oauth4os is additive** — direct OpenSearch access still works throughout migration
2. Point clients back to OpenSearch directly (update DNS/config)
3. Remove oauth4os from the network path
4. No OpenSearch configuration changes needed to roll back

The proxy never modifies OpenSearch's security configuration. Rolling back is always safe.

---

## Checklist

- [ ] Deploy oauth4os alongside existing setup
- [ ] Create OpenSearch backend roles matching your scopes
- [ ] Register clients and test token issuance
- [ ] Add Cedar policies (optional)
- [ ] Add rate limits (optional)
- [ ] Migrate clients one at a time
- [ ] Verify audit logs show expected access patterns
- [ ] Restrict direct OpenSearch access to proxy IP only
- [ ] Decommission old auth method
