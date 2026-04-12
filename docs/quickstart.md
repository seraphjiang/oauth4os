# Developer Quickstart — oauth4os in 5 Minutes

## Fastest Start: Docker Hub

```bash
docker run -p 8443:8443 jianghuan/oauth4os:latest
# Open http://localhost:8443
```

That's it — the proxy is running with a built-in demo client. Skip to [Step 2](#step-2-get-a-token) to start using it.

## Full Stack: Proxy + OpenSearch + Keycloak

For a complete setup with OpenSearch and an OIDC provider:

```
Developer → oauth4os (:8443) → OpenSearch (:9200)
                ↕                    ↕
           Keycloak (:8080)    Dashboards (:5601)
```

## Step 1: Start Everything

```bash
git clone https://github.com/seraphjiang/oauth4os.git
cd oauth4os
docker compose -f docker-compose.demo.yml up -d
```

Wait ~60 seconds for OpenSearch + Keycloak to initialize.

```bash
# Verify everything is running
curl -s http://localhost:8443/health | jq
# → {"status":"ok","version":"1.0.0"}
```

## Step 2: Get a Scoped Token

Keycloak comes pre-configured with two clients:

| Client | Scope | Can do |
|--------|-------|--------|
| `log-reader` | `read:logs-*` | Search logs indices only |
| `admin-agent` | `admin` | Full access |

```bash
# Get a read-only token for log-reader
TOKEN=$(curl -s -X POST http://localhost:8080/realms/opensearch/protocol/openid-connect/token \
  -d "grant_type=client_credentials" \
  -d "client_id=log-reader" \
  -d "client_secret=log-reader-secret" \
  -d "scope=read:logs-*" | jq -r '.access_token')

echo $TOKEN
```

## Step 3: Use the Token

```bash
# Create some test data (using admin token — log-reader can't write)
ADMIN_TOKEN=$(curl -s -X POST http://localhost:8080/realms/opensearch/protocol/openid-connect/token \
  -d "grant_type=client_credentials" \
  -d "client_id=admin-agent" \
  -d "client_secret=admin-agent-secret" \
  -d "scope=admin" | jq -r '.access_token')

# Index a log entry
curl -X POST "http://localhost:8443/logs-2025.01.15/_doc" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"timestamp":"2025-01-15T10:30:00Z","level":"error","service":"payment-api","message":"Connection refused to database"}'

# Search with the read-only token — this works
curl -s "http://localhost:8443/logs-*/_search" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":{"match":{"level":"error"}}}' | jq '.hits.total'

# Try to delete with read-only token — this is BLOCKED
curl -s -X DELETE "http://localhost:8443/logs-2025.01.15" \
  -H "Authorization: Bearer $TOKEN"
# → 403 Forbidden (scope doesn't allow delete)
```

## Step 4: See the Audit Trail

```bash
# Every request through the proxy is logged
docker logs oauth4os-proxy 2>&1 | grep "client=log-reader"
# → [2025-01-15T10:31:00Z] client=log-reader scopes=[read:logs-*] GET /logs-*/_search
```

## Step 5: Use the CLI

```bash
# Install (or use from source)
go install github.com/seraphjiang/oauth4os/cmd/cli@latest

# Login
oauth4os login log-reader log-reader-secret "read:logs-*"
# → ✅ Authenticated. Token cached (~/.oauth4os.yaml)

# Search
curl -H "Authorization: Bearer $(oauth4os status --token-only)" \
  "http://localhost:8443/logs-*/_search?q=level:error"

# Check status
oauth4os status
# → Logged in as: log-reader | Scopes: read:logs-* | Expires: 59m
```

## Step 6: Connect an AI Agent

```json
// Claude Desktop — ~/.config/claude/claude_desktop_config.json
{
  "mcpServers": {
    "opensearch": {
      "command": "python",
      "args": ["examples/mcp-server/server.py"],
      "env": {
        "OAUTH4OS_URL": "http://localhost:8443",
        "OAUTH4OS_TOKEN_URL": "http://localhost:8080/realms/opensearch/protocol/openid-connect/token",
        "OAUTH4OS_CLIENT_ID": "log-reader",
        "OAUTH4OS_CLIENT_SECRET": "log-reader-secret",
        "OAUTH4OS_SCOPE": "read:logs-*"
      }
    }
  }
}
```

Then ask Claude: "Search for all errors in the payment service"

## Step 7: Add Cedar Policies (Optional)

```yaml
# config.yaml — add fine-grained policies
cedar:
  enabled: true
  policies:
    - "permit(*, GET, logs-*) when { principal.scope contains \"read:logs\" };"
    - "forbid(*, *, .opendistro_security);"
    - "forbid(*, DELETE, *) unless { principal.role == \"admin\" };"
```

## What Just Happened?

1. **Keycloak** issued a JWT with `scope: read:logs-*`
2. **oauth4os** validated the JWT via JWKS, mapped `read:logs-*` → `logs_read_access` role
3. **Cedar** evaluated the request against policies
4. **OpenSearch** received the request with the mapped role — FGAC enforced
5. **Audit log** recorded who did what

No OpenSearch credentials were shared. The token expires. It can be revoked. Every request is audited.

## Next Steps

- [Configuration Guide](deployment.md) — all YAML options
- [Deployment Guide](deployment.md) — production setup on K8s/ECS
- [MCP Integration](../examples/mcp-server/README.md) — AI agent setup
- [Cedar Policies](cedar-guide.md) — fine-grained access control
