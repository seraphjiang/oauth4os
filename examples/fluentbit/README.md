# Fluent Bit → oauth4os → OpenSearch

Ship logs from Fluent Bit through oauth4os with scoped write access.

## Architecture

```
┌────────────┐     ┌──────────────┐     ┌─────────────────┐
│ Fluent Bit │────▶│  oauth4os    │────▶│   OpenSearch     │
│ (logs)     │     │  proxy       │     │   (storage)      │
│            │     │  Bearer auth │     │                  │
└────────────┘     └──────────────┘     └─────────────────┘
```

## Setup

1. Create a scoped token for Fluent Bit:

```bash
curl -X POST http://localhost:8443/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=fluentbit-prod" \
  -d "client_secret=<secret>" \
  -d "scope=write:logs-*"
```

2. Configure Fluent Bit output:

```ini
# fluent-bit.conf
[OUTPUT]
    Name            opensearch
    Match           *
    Host            oauth4os-proxy
    Port            8443
    Index           logs-app
    HTTP_User       ""
    HTTP_Passwd     ""
    # oauth4os token auth via header
    Header          Authorization Bearer <token>
    Suppress_Type_Name On
    tls             Off
```

3. Start with Docker Compose:

```bash
docker compose up -d
```

## Docker Compose

See `docker-compose.yml` in this directory for a complete working setup.

## Scope Enforcement

The `write:logs-*` scope allows writing to any `logs-*` index. Attempts to write to other indices (e.g., `metrics-*`) will be denied by Cedar policy.

## Token Rotation

For production, use the CLI to rotate tokens:

```bash
oauth4os-cli create-token --client-id fluentbit-prod --scope "write:logs-*" --output json
# Update Fluent Bit config with new token
# Revoke old token
oauth4os-cli revoke-token --id <old-token-id>
```
