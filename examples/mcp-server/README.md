# MCP Server Example — AI Agent → oauth4os → OpenSearch

A [Model Context Protocol](https://modelcontextprotocol.io/) server that gives AI agents secure, scoped access to OpenSearch through oauth4os.

## How It Works

```
┌──────────┐     ┌────────────┐     ┌──────────┐     ┌────────────┐
│ AI Agent │────▶│ MCP Server │────▶│ oauth4os │────▶│ OpenSearch │
│ (Claude) │ MCP │ (Python)   │ HTTP│ (proxy)  │     │ (Engine)   │
└──────────┘     └────────────┘     └──────────┘     └────────────┘
                   Scoped token        JWT validated
                   auto-refreshed      scope→role mapped
```

The MCP server:
1. Authenticates to oauth4os using client_credentials grant
2. Caches and auto-refreshes the access token
3. Exposes 3 tools to the AI agent: `search_logs`, `get_indices`, `get_cluster_health`
4. All requests go through oauth4os — scoped, audited, revocable

## Quick Start

```bash
# Set environment
export OAUTH4OS_URL=http://localhost:8443
export OAUTH4OS_CLIENT_ID=mcp-agent
export OAUTH4OS_CLIENT_SECRET=secret
export OAUTH4OS_SCOPE="read:logs-*"

# Run
python3 server.py
```

## Add to Claude Desktop

```json
{
  "mcpServers": {
    "opensearch": {
      "command": "python3",
      "args": ["/path/to/server.py"],
      "env": {
        "OAUTH4OS_URL": "http://localhost:8443",
        "OAUTH4OS_CLIENT_ID": "mcp-agent",
        "OAUTH4OS_CLIENT_SECRET": "secret",
        "OAUTH4OS_SCOPE": "read:logs-*"
      }
    }
  }
}
```

## Tools

| Tool | Description |
|------|-------------|
| `search_logs` | Search OpenSearch index with a query string |
| `get_indices` | List available indices |
| `get_cluster_health` | Check OpenSearch cluster status |

## Security

- Token is scoped to `read:logs-*` — agent can only read log indices
- Token auto-refreshes before expiry (no long-lived credentials)
- All requests audited by oauth4os
- Token can be revoked at any time via oauth4os API
