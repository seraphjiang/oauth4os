# MCP Server — AI Agent → oauth4os → OpenSearch

Reference [Model Context Protocol](https://modelcontextprotocol.io/) server that gives AI agents secure, scoped access to OpenSearch through oauth4os.

## How It Works

```
┌──────────┐     ┌────────────┐     ┌──────────┐     ┌────────────┐
│ AI Agent │────▶│ MCP Server │────▶│ oauth4os │────▶│ OpenSearch │
│ (Claude) │ MCP │ (Python)   │ HTTP│ (proxy)  │     │ (Engine)   │
└──────────┘     └────────────┘     └──────────┘     └────────────┘
                   Scoped token        JWT validated
                   auto-refreshed      scope→role mapped
```

## Quick Start

```bash
pip install -r requirements.txt

export OAUTH4OS_URL=http://localhost:8443
export OAUTH4OS_CLIENT_ID=mcp-agent
export OAUTH4OS_CLIENT_SECRET=secret
export OAUTH4OS_SCOPE="read:logs-* write:logs-*"

# MCP mode (for Claude Desktop)
python3 server.py

# Standalone mode (for testing)
python3 server.py get_indices
python3 server.py search_logs '{"index": "logs-*", "query": "level:error"}'
python3 server.py aggregate '{"index": "logs-*", "field": "service.keyword", "agg_type": "terms"}'
```

## Claude Desktop Config

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
        "OAUTH4OS_SCOPE": "read:logs-* write:logs-*"
      }
    }
  }
}
```

## Tools (7)

| Tool | Scope | Description |
|------|-------|-------------|
| `search_logs` | read | Search index with Lucene query string |
| `aggregate` | read | Run aggregations: terms, date_histogram, avg, sum, min, max, percentiles |
| `get_indices` | read | List indices with doc counts and sizes |
| `get_mappings` | read | Get field names and types for an index |
| `create_index` | write | Create index with optional mappings |
| `delete_docs` | write | Delete documents matching a query |
| `get_cluster_health` | read | Cluster status, node count, shard info |

## Example Conversations

**"Show me error trends"**
→ Agent calls `aggregate` with `{"index": "logs-*", "field": "@timestamp", "agg_type": "date_histogram", "query": "level:error"}`

**"What fields are in my logs?"**
→ Agent calls `get_mappings` with `{"index": "logs-*"}`

**"Clean up old test data"**
→ Agent calls `delete_docs` with `{"index": "test-logs", "query": "environment:test AND @timestamp:<2025-01-01"}`

## Security

- Token scoped via `OAUTH4OS_SCOPE` — `read:logs-*` can only read log indices
- Token auto-refreshes 60s before expiry
- All requests audited by oauth4os proxy
- Token revocable at any time via `oauth4os revoke <token_id>`
- Write operations require explicit `write:` scope
