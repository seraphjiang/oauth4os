# Operator Runbook

Common operational tasks and incident response procedures for oauth4os.

---

## Health Checks

```bash
# Quick check
curl -sf https://proxy:8443/health | jq .

# Deep check (upstream, JWKS, TLS)
curl -sf https://proxy:8443/health/deep | jq .

# Readiness (Kubernetes)
curl -sf https://proxy:8443/ready | jq .

# Version
curl -sf https://proxy:8443/version | jq .
```

---

## Incident: High Error Rate

**Symptom**: `oauth4os_requests_failed` increasing, users reporting 5xx errors.

1. Check upstream health:
   ```bash
   curl -sf https://proxy:8443/health/deep | jq .upstream
   ```
2. If upstream is down → OpenSearch issue, not proxy. Check OpenSearch logs.
3. If upstream is healthy → check auth failures:
   ```bash
   curl -sf https://proxy:8443/metrics | grep auth_failed
   ```
4. If auth failures spiking → likely expired JWKS or IdP issue:
   ```bash
   curl -sf https://proxy:8443/health/deep | jq .jwks
   ```
5. Check recent audit log for patterns:
   ```bash
   curl -sf "https://proxy:8443/admin/audit?limit=20" | jq '.[] | {client_id, status, path}'
   ```

---

## Incident: Rate Limiting Spike

**Symptom**: `oauth4os_rate_limited` increasing, clients getting 429.

1. Identify which clients are hitting limits:
   ```bash
   curl -sf https://proxy:8443/admin/analytics | jq '.top_clients[:5]'
   ```
2. Increase limits temporarily:
   ```bash
   # Check current limits
   curl -sf https://proxy:8443/admin/rate-limits

   # Increase for a specific scope
   curl -X PUT https://proxy:8443/admin/rate-limits \
     -H "Content-Type: application/json" \
     -d '{"read:logs-*": 1200}'
   ```
3. Investigate if the client is misbehaving or if limits are too low for legitimate traffic.

---

## Incident: JWKS Refresh Failure

**Symptom**: All auth failing, `auth_failed` spiking.

1. Check JWKS status:
   ```bash
   curl -sf https://proxy:8443/health/deep | jq .jwks
   ```
2. Verify IdP is reachable from proxy:
   ```bash
   # From proxy container
   curl -sf https://keycloak.example.com/realms/opensearch/.well-known/openid-configuration
   ```
3. If IdP is down → proxy uses cached JWKS until it expires. Tokens already issued continue working.
4. If IdP is up but JWKS fails → check TLS certs, DNS, network policies.
5. Restart proxy to force JWKS refresh (last resort):
   ```bash
   # App Runner
   aws apprunner start-deployment --service-arn <arn>
   # Kubernetes
   kubectl rollout restart deployment oauth4os
   ```

---

## Token Operations

```bash
# List active tokens
curl -sf https://proxy:8443/oauth/tokens | jq '.[].client_id'

# Revoke a specific token
curl -X DELETE https://proxy:8443/oauth/token/<token-id>

# Force logout all sessions for a client
curl -X POST https://proxy:8443/admin/sessions/logout \
  -H "Content-Type: application/json" \
  -d '{"client_id": "compromised-agent"}'

# Rotate a client secret (invalidates old secret immediately)
curl -X POST https://proxy:8443/oauth/register/<client_id>/rotate
```

---

## Client Management

```bash
# List all clients
curl -sf https://proxy:8443/oauth/register | jq '.[].client_name'

# Register a new client
curl -X POST https://proxy:8443/oauth/register \
  -H "Content-Type: application/json" \
  -d '{"client_name":"new-agent","scope":"read:logs-*"}'

# Delete a client (revokes all its tokens)
curl -X DELETE https://proxy:8443/oauth/register/<client_id>
```

---

## Config Backup/Restore

```bash
# Export current config
curl -sf https://proxy:8443/admin/backup > backup.json

# Restore config (careful — overwrites current)
curl -X POST https://proxy:8443/admin/restore \
  -H "Content-Type: application/json" \
  -d @backup.json
```

---

## Log Analysis

```bash
# Recent audit entries
curl -sf "https://proxy:8443/admin/audit?limit=50" | jq .

# Filter by client
curl -sf "https://proxy:8443/admin/audit?client_id=suspicious-agent&limit=100" | jq .

# Check analytics
curl -sf https://proxy:8443/admin/analytics | jq .
```

---

## Restart Procedures

**App Runner**:
```bash
aws apprunner start-deployment --service-arn <arn> --region us-west-2
```

**Kubernetes**:
```bash
kubectl rollout restart deployment oauth4os
kubectl rollout status deployment oauth4os
```

**Docker**:
```bash
docker restart oauth4os
```

The proxy handles graceful shutdown — in-flight requests drain for up to 30 seconds before termination. The `/ready` endpoint returns 503 during shutdown so load balancers stop routing new traffic.

---

## Scaling

```bash
# App Runner — update instance count
aws apprunner update-service --service-arn <arn> \
  --auto-scaling-configuration-arn <asc-arn>

# Kubernetes
kubectl scale deployment oauth4os --replicas=3

# Check current load
curl -sf https://proxy:8443/metrics | grep requests_active
```

The proxy is stateless — scale horizontally without coordination. Tokens are self-contained JWTs validated locally.
