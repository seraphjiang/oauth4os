# Canary Deployment Guide

oauth4os supports canary deployments by routing a percentage of traffic to a new version.

## Strategy

Use the proxy's multi-cluster federation to split traffic:

```yaml
# config.yaml — canary: 10% to v2, 90% to v1
clusters:
  stable:
    engine: https://opensearch-v1.internal:9200
    prefixes: ["*"]
  canary:
    engine: https://opensearch-v2.internal:9200
    prefixes: ["canary-*"]
```

## AppRunner Canary

AppRunner doesn't support weighted routing natively. Use Route 53 weighted records:

```bash
# 90% to stable
aws route53 change-resource-record-sets --hosted-zone-id Z123 \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "api.oauth4os.example.com",
        "Type": "CNAME",
        "SetIdentifier": "stable",
        "Weight": 90,
        "TTL": 60,
        "ResourceRecords": [{"Value": "stable.us-west-2.awsapprunner.com"}]
      }
    }]
  }'

# 10% to canary
aws route53 change-resource-record-sets --hosted-zone-id Z123 \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "api.oauth4os.example.com",
        "Type": "CNAME",
        "SetIdentifier": "canary",
        "Weight": 10,
        "TTL": 60,
        "ResourceRecords": [{"Value": "canary.us-west-2.awsapprunner.com"}]
      }
    }]
  }'
```

## Kubernetes Canary

With the Helm chart:

```yaml
# values-canary.yaml
replicaCount: 1
image:
  tag: v0.5.0-rc1

ingress:
  annotations:
    nginx.ingress.kubernetes.io/canary: "true"
    nginx.ingress.kubernetes.io/canary-weight: "10"
```

```bash
helm upgrade oauth4os-canary deploy/helm \
  -f values-canary.yaml \
  --set image.tag=v0.5.0-rc1
```

## Health-Based Rollback

Monitor the canary with `/health/deep` and `/metrics`:

```bash
# Check canary health
curl -s https://canary.oauth4os.example.com/health/deep | jq .overall

# Compare error rates
curl -s https://stable.oauth4os.example.com/metrics | grep oauth4os_requests_failed
curl -s https://canary.oauth4os.example.com/metrics | grep oauth4os_requests_failed
```

If the canary shows elevated errors, roll back by setting canary weight to 0.
