# Kubernetes Sidecar Pattern — oauth4os

Run oauth4os as a sidecar container alongside your application. The app talks to `localhost:8443` — no token management in app code.

## Architecture

```
┌─── Pod ──────────────────────────────┐
│                                      │
│  ┌──────────┐    ┌──────────────┐    │     ┌─────────────┐
│  │   App    │───▶│  oauth4os    │────│────▶│ OpenSearch   │
│  │ :8080    │    │  sidecar     │    │     │ cluster      │
│  │          │    │  :8443       │    │     │              │
│  └──────────┘    └──────────────┘    │     └─────────────┘
│                                      │
└──────────────────────────────────────┘
```

## Why Sidecar?

- App sends plain HTTP to `localhost:8443` — no OAuth logic needed
- Token management handled by the sidecar
- Scopes enforced at the proxy level
- One config change to rotate credentials (restart sidecar, not app)

## Deploy

```bash
# Using Helm
helm install oauth4os-sidecar ./helm \
  --set opensearch.url=https://opensearch.internal:9200 \
  --set oauth.clientId=my-app \
  --set oauth.clientSecret=secret \
  --set oauth.scope="read:logs-*"

# Or plain kubectl
kubectl apply -f deployment.yaml
```

## Files

- `deployment.yaml` — Pod with app + oauth4os sidecar
- `configmap.yaml` — oauth4os config
- `helm/` — Helm chart for reusable deployment
