# Secret Management Guide

How to manage secrets in oauth4os for production deployments.

## Current State (v1.0.0)

Secrets are stored in `config.yaml` or environment variables:

```yaml
providers:
  - name: keycloak
    client_secret: my-secret-value  # ⚠️ plaintext in config
```

## Recommended: Environment Variables

Override any config value with environment variables:

```bash
docker run -p 8443:8443 \
  -e OAUTH4OS_PROVIDER_CLIENT_SECRET=my-secret \
  -e OAUTH4OS_SIGNING_KEY_FILE=/run/secrets/signing-key \
  jianghuan/oauth4os:latest
```

## Docker Secrets (Compose)

```yaml
services:
  oauth-proxy:
    image: jianghuan/oauth4os:latest
    secrets:
      - provider_secret
      - signing_key
    environment:
      - OAUTH4OS_PROVIDER_CLIENT_SECRET_FILE=/run/secrets/provider_secret

secrets:
  provider_secret:
    file: ./secrets/provider_secret.txt
  signing_key:
    file: ./secrets/signing_key.pem
```

## Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: oauth4os-secrets
type: Opaque
stringData:
  client-secret: "my-secret-value"
  signing-key: |
    -----BEGIN EC PRIVATE KEY-----
    ...
    -----END EC PRIVATE KEY-----
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
        - name: oauth4os
          envFrom:
            - secretRef:
                name: oauth4os-secrets
```

## What NOT to Do

| ❌ Don't | ✅ Do Instead |
|----------|--------------|
| Commit secrets to git | Use env vars or secret mounts |
| Use plaintext in config.yaml | Use `_FILE` suffix for file-based secrets |
| Share secrets across environments | Separate secrets per env |
| Use long-lived signing keys | Enable key rotation (default: 24h) |

## Key Rotation

oauth4os rotates signing keys automatically:

```yaml
keyring:
  rotation_interval: 24h  # default
  max_keys: 3             # keep 3 keys for graceful rollover
```

The JWKS endpoint (`/.well-known/jwks.json`) always serves current + previous keys so tokens signed with the old key remain valid during rotation.
