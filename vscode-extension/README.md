# oauth4os VS Code Extension

Manage OAuth tokens, inspect scopes, and query OpenSearch directly from VS Code.

## Features

- **Token sidebar** — activity bar panel showing active tokens with status indicators (🟢 active, 🟡 expired, 🔴 revoked)
- **Create token** — command palette: client ID, secret, scope → token copied to clipboard
- **Revoke token** — right-click token in sidebar → confirm → revoked
- **Inspect token** — click token → opens JSON document with full details
- **Query OpenSearch** — run queries through the proxy with your active token
- **Metrics view** — live Prometheus metrics from the proxy
- **Auto-connect** — connects to proxy on activation, shows version in status bar

## Commands

| Command | Description |
|---------|-------------|
| `oauth4os: Connect to Proxy` | Set proxy URL and test connection |
| `oauth4os: Create Token` | Issue a new scoped token |
| `oauth4os: Revoke Token` | Revoke an active token |
| `oauth4os: Inspect Token` | View token details as JSON |
| `oauth4os: Query OpenSearch` | Run a query through the proxy |
| `oauth4os: Copy Token` | Copy token ID to clipboard |
| `oauth4os: Refresh Token List` | Reload the token sidebar |

## Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `oauth4os.proxyUrl` | `http://localhost:8443` | Proxy URL |
| `oauth4os.clientId` | — | Default client ID for token creation |
| `oauth4os.defaultScope` | `read:logs-*` | Default scope for new tokens |

## Sidebar Views

- **Tokens** — list of all tokens with status, scopes, click to inspect
- **Scope Mappings** — proxy info (URL, version, uptime)
- **Audit Log** — live Prometheus metrics

## Development

```bash
cd vscode-extension
npm install  # no deps needed — uses built-in vscode API + fetch
# Press F5 in VS Code to launch Extension Development Host
```

## Publishing

```bash
npx vsce package
npx vsce publish
```
