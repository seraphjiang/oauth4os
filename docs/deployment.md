# Deployment Guide

This guide covers deploying oauth4os to AWS App Runner, ECS, Kubernetes, and local Docker. Each section includes IAM configuration, AOSS setup, and config examples.

---

## Prerequisites

All deployment targets need:

1. **Docker image** — built from the included Dockerfile
2. **config.yaml** — proxy configuration (upstream, providers, scopes)
3. **IAM credentials** (if using AOSS) — role with `aoss:APIAccessAll` permission

### Build the Docker image

```bash
docker build -t oauth4os:latest .
```

### AOSS Collection Setup

If targeting OpenSearch Serverless:

1. Create a collection in the AWS Console (type: Search or Time Series)
2. Create a data access policy granting your IAM role access:

```json
[
  {
    "Rules": [
      {
        "Resource": ["collection/your-collection"],
        "Permission": ["aoss:*"],
        "ResourceType": "collection"
      },
      {
        "Resource": ["index/your-collection/*"],
        "Permission": ["aoss:*"],
        "ResourceType": "index"
      }
    ],
    "Principal": ["arn:aws:iam::ACCOUNT:role/oauth4os-role"]
  }
]
```

3. Note the collection endpoint: `https://<id>.<region>.aoss.amazonaws.com`

---

## Option 1: AWS App Runner

Best for: quick deployment, auto-scaling, no infrastructure to manage.

### IAM Role

Create an instance role for App Runner with AOSS access:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "aoss:APIAccessAll",
      "Resource": "arn:aws:aoss:<region>:<account>:collection/<collection-id>"
    }
  ]
}
```

Trust policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {"Service": "tasks.apprunner.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }
  ]
}
```

### config.yaml

```yaml
upstream:
  engine: https://<collection-id>.<region>.aoss.amazonaws.com
  sigv4:
    region: us-west-2
    service: aoss

providers:
  - name: self
    issuer: https://<your-apprunner-url>
    jwks_uri: auto

scope_mapping:
  "read:logs-*":
    backend_roles: [logs_read_access]
  "admin":
    backend_roles: [all_access]

listen: :8443
```

### New in v1.1.0: Secrets & Persistence

```yaml
# Secret references — resolve from env vars or files instead of plaintext
# Supported schemes: env:VAR_NAME, file:/path/to/secret
webhook:
  url: env:WEBHOOK_URL          # reads from $WEBHOOK_URL at startup

# Token persistence — survive restarts (default: memory)
token_store: file:/var/lib/oauth4os/tokens.json

# Secrets backend (default: env)
secrets_backend: env             # env or file
```

### Deploy

```bash
# Push to ECR
aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin <account>.dkr.ecr.us-west-2.amazonaws.com
docker tag oauth4os:latest <account>.dkr.ecr.us-west-2.amazonaws.com/oauth4os:latest
docker push <account>.dkr.ecr.us-west-2.amazonaws.com/oauth4os:latest

# Create App Runner service
aws apprunner create-service \
  --service-name oauth4os \
  --source-configuration '{
    "ImageRepository": {
      "ImageIdentifier": "<account>.dkr.ecr.us-west-2.amazonaws.com/oauth4os:latest",
      "ImageRepositoryType": "ECR",
      "ImageConfiguration": {
        "Port": "8443"
      }
    },
    "AutoDeploymentsEnabled": true,
    "AuthenticationConfiguration": {
      "AccessRoleArn": "arn:aws:iam::<account>:role/apprunner-ecr-access"
    }
  }' \
  --instance-configuration '{
    "InstanceRoleArn": "arn:aws:iam::<account>:role/oauth4os-role",
    "Cpu": "0.25 vCPU",
    "Memory": "0.5 GB"
  }' \
  --health-check-configuration '{
    "Protocol": "HTTP",
    "Path": "/health",
    "Interval": 10,
    "Timeout": 5,
    "HealthyThreshold": 1,
    "UnhealthyThreshold": 3
  }' \
  --region us-west-2
```

**Auto-deploy**: With `AutoDeploymentsEnabled: true`, pushing a new image to ECR triggers automatic redeployment.

**Cost**: ~$5-15/month for a single instance (0.25 vCPU, 0.5 GB).

---

## Option 2: Amazon ECS (Fargate)

Best for: production workloads, fine-grained networking, service mesh integration.

### IAM Task Role

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "aoss:APIAccessAll",
      "Resource": "arn:aws:aoss:<region>:<account>:collection/<collection-id>"
    }
  ]
}
```

Trust policy — use `ecs-tasks.amazonaws.com`.

### Task Definition

```json
{
  "family": "oauth4os",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "taskRoleArn": "arn:aws:iam::<account>:role/oauth4os-task-role",
  "executionRoleArn": "arn:aws:iam::<account>:role/ecsTaskExecutionRole",
  "containerDefinitions": [
    {
      "name": "oauth4os",
      "image": "<account>.dkr.ecr.<region>.amazonaws.com/oauth4os:latest",
      "portMappings": [{"containerPort": 8443, "protocol": "tcp"}],
      "healthCheck": {
        "command": ["CMD-SHELL", "wget -qO- http://localhost:8443/health || exit 1"],
        "interval": 10,
        "timeout": 5,
        "retries": 3
      },
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/oauth4os",
          "awslogs-region": "<region>",
          "awslogs-stream-prefix": "proxy"
        }
      }
    }
  ]
}
```

### Service with ALB

```bash
# Create service behind an ALB
aws ecs create-service \
  --cluster default \
  --service-name oauth4os \
  --task-definition oauth4os \
  --desired-count 2 \
  --launch-type FARGATE \
  --network-configuration '{
    "awsvpcConfiguration": {
      "subnets": ["subnet-xxx"],
      "securityGroups": ["sg-xxx"],
      "assignPublicIp": "ENABLED"
    }
  }' \
  --load-balancers '[{
    "targetGroupArn": "arn:aws:elasticloadbalancing:...",
    "containerName": "oauth4os",
    "containerPort": 8443
  }]'
```

**Scaling**: Add an auto-scaling policy targeting CPU utilization or request count.

**Cost**: ~$15-30/month for 2 tasks (0.25 vCPU, 0.5 GB each).

---

## Option 3: Kubernetes (Helm)

Best for: teams already running Kubernetes, multi-cluster setups.

### IAM (EKS)

Use IAM Roles for Service Accounts (IRSA):

```bash
eksctl create iamserviceaccount \
  --name oauth4os \
  --namespace default \
  --cluster my-cluster \
  --attach-policy-arn arn:aws:iam::<account>:policy/oauth4os-aoss-access \
  --approve
```

### Helm Install

```bash
helm install oauth4os deploy/helm/oauth4os/ \
  --set image.repository=<account>.dkr.ecr.<region>.amazonaws.com/oauth4os \
  --set image.tag=latest \
  --set config.upstream.engine=https://<collection-id>.<region>.aoss.amazonaws.com \
  --set config.upstream.sigv4.region=us-west-2 \
  --set config.upstream.sigv4.service=aoss \
  --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"=arn:aws:iam::<account>:role/oauth4os-role
```

### values.yaml

```yaml
replicaCount: 2

image:
  repository: <account>.dkr.ecr.<region>.amazonaws.com/oauth4os
  tag: latest
  pullPolicy: Always

service:
  type: ClusterIP
  port: 8443

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: oauth4os.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: oauth4os-tls
      hosts:
        - oauth4os.example.com

config:
  upstream:
    engine: https://<collection-id>.<region>.aoss.amazonaws.com
    sigv4:
      region: us-west-2
      service: aoss
  providers:
    - name: keycloak
      issuer: https://keycloak.example.com/realms/opensearch
      jwks_uri: auto
  scope_mapping:
    "read:logs-*":
      backend_roles: [logs_read_access]
    "admin":
      backend_roles: [all_access]
  listen: :8443

resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 256Mi

serviceAccount:
  create: true
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::<account>:role/oauth4os-role

livenessProbe:
  httpGet:
    path: /health
    port: 8443
  initialDelaySeconds: 5

readinessProbe:
  httpGet:
    path: /health
    port: 8443
  initialDelaySeconds: 3
```

**Cost**: Depends on cluster — the proxy itself uses ~50MB RAM and minimal CPU.

---

## Option 4: Local Docker

Best for: development, testing, demos.

### Standalone (proxy only)

```bash
docker run -d \
  --name oauth4os \
  -p 8443:8443 \
  -v $(pwd)/config.yaml:/etc/oauth4os/config.yaml \
  oauth4os:latest
```

### With OpenSearch (docker-compose)

```bash
docker compose up
```

The included `docker-compose.yml` starts:
- oauth4os proxy on `:8443`
- OpenSearch Engine on `:9200`
- OpenSearch Dashboards on `:5601`

### config.yaml for local OpenSearch

```yaml
upstream:
  engine: https://opensearch:9200
  dashboards: https://opensearch-dashboards:5601

providers:
  - name: self
    issuer: http://localhost:8443
    jwks_uri: auto

scope_mapping:
  "read:logs-*":
    backend_roles: [logs_read_access]
  "admin":
    backend_roles: [all_access]

listen: :8443

tls:
  insecure_skip_verify: true
```

### Seed demo data

```bash
PROXY_URL=http://localhost:8443 ./scripts/seed-demo.sh
```

---

## Environment Variables

The proxy reads these environment variables (override config.yaml):

| Variable | Description | Default |
|----------|-------------|---------|
| `OAUTH4OS_LISTEN` | Listen address | `:8443` |
| `OAUTH4OS_UPSTREAM` | OpenSearch URL | from config |
| `OAUTH4OS_LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `AWS_REGION` | AWS region for SigV4 | from config |
| `AWS_ACCESS_KEY_ID` | AWS credentials (if not using IAM role) | — |
| `AWS_SECRET_ACCESS_KEY` | AWS credentials | — |
| `AWS_SESSION_TOKEN` | AWS session token (temporary creds) | — |

---

## Health Checks

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Basic health — returns `{"status":"ok","version":"1.0.0"}` |
| `GET /health/deep` | Deep health — checks upstream connectivity, JWKS freshness, TLS cert expiry |

Use `/health` for load balancer health checks (fast, no external calls). Use `/health/deep` for monitoring dashboards.

---

## TLS Configuration

### Proxy TLS (client → proxy)

```yaml
tls:
  enabled: true
  cert_file: /etc/oauth4os/tls.crt
  key_file: /etc/oauth4os/tls.key
```

### Upstream TLS (proxy → OpenSearch)

For self-signed OpenSearch certificates:

```yaml
tls:
  insecure_skip_verify: true  # development only
```

For production, add the CA to the system trust store or mount it in the container.

### Mutual TLS (client certificates)

```yaml
mtls:
  enabled: true
  ca_file: /etc/oauth4os/client-ca.pem
  clients:
    "agent.example.com":
      client_id: agent
      scopes: [read:logs-*]
```

---

## Monitoring

### Prometheus

Scrape `GET /metrics` on port 8443. Example scrape config:

```yaml
scrape_configs:
  - job_name: oauth4os
    static_configs:
      - targets: ['oauth4os:8443']
    metrics_path: /metrics
    scheme: https
    tls_config:
      insecure_skip_verify: true
```

### Audit Logs

JSON audit logs go to stdout. In ECS/Kubernetes, these are captured by the log driver automatically. Query via the Admin API:

```bash
curl https://oauth4os.example.com/admin/audit?client_id=my-agent&limit=100
```

### Analytics Dashboard

Visit `/developer/analytics` for a live dashboard showing top clients, scope distribution, and request timelines.
