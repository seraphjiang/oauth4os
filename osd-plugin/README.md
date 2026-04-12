# oauth4os — OSD Plugin (Token Management UI)

OpenSearch Dashboards plugin for managing OAuth tokens issued by the oauth4os proxy.

## Features

- **List tokens** — view all active tokens with client, scopes, status, creation time
- **Create token** — issue new scoped tokens via client credentials grant
- **Revoke token** — revoke active tokens with confirmation dialog
- **Copy to clipboard** — one-click copy for access and refresh tokens

## Structure

```
osd-plugin/
  opensearch_dashboards.json  — OSD plugin manifest
  public/
    index.html                — Standalone test harness
    token-management.js       — React components (TokenManagement, TokenList, CreateTokenForm)
    token-management.css      — Styles (dark theme, responsive)
```

## Standalone Testing

1. Start the oauth4os proxy: `docker compose up`
2. Open `osd-plugin/public/index.html` in a browser
3. The UI connects to `http://localhost:8443/oauth` by default

## Integration with OSD

The `TokenManagement` component is exported as `window.OAuth4osTokenManagement`. To mount inside an OSD plugin:

```js
import { TokenManagement } from './public/token-management';
// or
const App = window.OAuth4osTokenManagement;
ReactDOM.render(<App />, container);
```

Set `window.OAUTH4OS_API` to override the API base URL (defaults to `/oauth`).

## API Contract

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/oauth/tokens` | GET | List active tokens |
| `/oauth/token` | POST | Issue token (form-encoded: grant_type, client_id, client_secret, scope) |
| `/oauth/token/{id}` | DELETE | Revoke token |
| `/oauth/token/{id}` | GET | Get token details |
