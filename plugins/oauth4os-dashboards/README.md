# oauth4os — OpenSearch Dashboards Plugin

Token management UI for oauth4os. Lists, creates, and revokes OAuth tokens from within OpenSearch Dashboards.

## Features

- **List tokens**: View all issued tokens with status, scope, expiry
- **Create token**: Issue new client_credentials tokens with scoped access
- **Revoke token**: Instantly revoke active tokens

## Setup

1. Set `OAUTH4OS_PROXY_URL` environment variable to your oauth4os proxy address
2. Install the plugin into your OpenSearch Dashboards instance:

```bash
cd opensearch-dashboards
./bin/opensearch-dashboards-plugin install file:///path/to/oauth4os-dashboards.zip
```

3. Restart Dashboards — the plugin appears under **Management → OAuth Token Management**

## Architecture

```
OSD Browser → OSD Server Plugin → oauth4os Proxy → OpenSearch
                (routes)            (JWT + Cedar)
```

The server plugin proxies requests to the oauth4os API endpoints:
- `GET /api/oauth4os/tokens` → `GET /oauth/tokens`
- `POST /api/oauth4os/tokens` → `POST /oauth/token`
- `DELETE /api/oauth4os/tokens/{id}` → `DELETE /oauth/token/{id}`
