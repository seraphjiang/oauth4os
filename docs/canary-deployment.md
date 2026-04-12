# Canary Deployment Guide

Deploy oauth4os with gradual traffic shifting to minimize risk.

## Strategy

```
v1 (stable) ──── 100% traffic
                    ↓ deploy v2
v1 (stable) ──── 90% traffic
v2 (canary) ──── 10% traffic
                    ↓ monitor 5 min
v1 (stable) ──── 50% traffic
v2 (canary) ──── 50% traffic
                    ↓ monitor 5 min
v2 (new stable) ─ 100% traffic
```

## Kubernetes (Helm)

### 1. Deploy canary alongside stable

```bash
# Stable (already running)
helm upgrade oauth4os deploy/helm/ \
  --set replicaCount=3 \
  --set image.tag=v0.4.0

# Canary (1 replica, same service)
helm upgrade oauth4os-canary deploy/helm/ \
  --set replicaCount=1 \
  --set image.tag=v0.5.0 \
  --set nameOverride=oauth4os-canary \
  --set service.enabled=false
```

### 2. Split traffic with Istio

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: oauth4os
spec:
  hosts: [oauth4os]
  http:
    - route:
        - destination:
            host: oauth4os
            subset: stable
          weight: 90
        - destination:
            host: oauth4os
            subset: canary
          weight: 10
```

### 3. Monitor canary metrics

```bash
# Compare error rates
curl -s http://prometheus:9090/api/v1/query \
  --data-urlencode 'query=rate(oauth4os_auth_failed[5m])' | jq

# Check p99 latency
curl -s http://prometheus:9090/api/v1/query \
  --data-urlencode 'query=histogram_quantile(0.99, rate(oauth4os_request_duration_seconds_bucket[5m]))' | jq
```

### 4. Promote or rollback

```bash
# Promote: scale canary to full, remove stable
helm upgrade oauth4os deploy/helm/ --set image.tag=v0.5.0
helm uninstall oauth4os-canary

# Rollback: remove canary
helm uninstall oauth4os-canary
```

## AppRunner

AppRunner doesn't support traffic splitting natively. Use Route 53 weighted routing:

```bash
# Create canary deployment
aws apprunner create-service \
  --service-name oauth4os-canary \
  --source-configuration '{"ImageRepository":{"ImageIdentifier":"ECR_URI:v0.5.0","ImageRepositoryType":"ECR"}}'

# Route 53 weighted records
aws route53 change-resource-record-sets --hosted-zone-id ZONE_ID --change-batch '{
  "Changes": [
    {"Action":"UPSERT","ResourceRecordSet":{"Name":"oauth4os.example.com","Type":"CNAME","SetIdentifier":"stable","Weight":90,"TTL":60,"ResourceRecords":[{"Value":"STABLE_URL"}]}},
    {"Action":"UPSERT","ResourceRecordSet":{"Name":"oauth4os.example.com","Type":"CNAME","SetIdentifier":"canary","Weight":10,"TTL":60,"ResourceRecords":[{"Value":"CANARY_URL"}]}}
  ]
}'
```

## Health Check Gates

Before promoting canary, verify:

| Check | Threshold | Command |
|-------|-----------|---------|
| Error rate | < 1% | `curl /metrics \| grep auth_failed` |
| p99 latency | < 50ms | `curl /metrics \| grep request_duration` |
| Health endpoint | 200 OK | `curl /health` |
| Token issuance | Working | `curl -X POST /oauth/token -d '...'` |

## Docker Compose

```bash
# Run both versions
docker compose up -d
docker compose -f docker-compose.canary.yml up -d

# Use nginx/haproxy for traffic splitting
# See deploy/nginx/canary.conf for example config
```
