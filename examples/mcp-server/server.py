#!/usr/bin/env python3
"""MCP Server — AI agent access to OpenSearch via oauth4os.

Reference integration: authenticates with client_credentials, exposes
7 tools over the Model Context Protocol (stdio transport).

Requires: pip install mcp httpx
"""

import json
import os
import sys
import time

import httpx

# ── oauth4os Token Client ──────────────────────────────────────────────────────

OAUTH4OS_URL = os.environ.get("OAUTH4OS_URL", "http://localhost:8443")
CLIENT_ID = os.environ.get("OAUTH4OS_CLIENT_ID", "mcp-agent")
CLIENT_SECRET = os.environ.get("OAUTH4OS_CLIENT_SECRET", "secret")
SCOPE = os.environ.get("OAUTH4OS_SCOPE", "read:logs-* write:logs-*")

_token_cache: dict = {}


def _get_token() -> str:
    """Get a valid access token, refreshing if expired."""
    if _token_cache.get("token") and _token_cache.get("expires_at", 0) > time.time() + 60:
        return _token_cache["token"]
    resp = httpx.post(
        f"{OAUTH4OS_URL}/oauth/token",
        data={"grant_type": "client_credentials", "client_id": CLIENT_ID,
              "client_secret": CLIENT_SECRET, "scope": SCOPE},
        timeout=10,
    )
    resp.raise_for_status()
    data = resp.json()
    _token_cache["token"] = data["access_token"]
    _token_cache["expires_at"] = time.time() + data.get("expires_in", 3600)
    return _token_cache["token"]


def _os_request(method: str, path: str, body: dict = None) -> dict:
    """Make an authenticated request to OpenSearch through oauth4os."""
    token = _get_token()
    resp = httpx.request(
        method, f"{OAUTH4OS_URL}{path}",
        headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
        content=json.dumps(body) if body else None,
        timeout=30,
    )
    resp.raise_for_status()
    return resp.json()


# ── Tool Definitions ───────────────────────────────────────────────────────────

TOOLS = [
    {
        "name": "search_logs",
        "description": "Search an OpenSearch index with a query string. Returns matching documents.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "index": {"type": "string", "description": "Index pattern (e.g. 'logs-*')"},
                "query": {"type": "string", "description": "Lucene query string"},
                "size": {"type": "integer", "description": "Max results (default 10)"},
            },
            "required": ["index", "query"],
        },
    },
    {
        "name": "aggregate",
        "description": "Run an aggregation query on an OpenSearch index. Supports terms, date_histogram, avg, sum, min, max, percentiles.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "index": {"type": "string", "description": "Index pattern"},
                "field": {"type": "string", "description": "Field to aggregate on"},
                "agg_type": {"type": "string", "description": "Aggregation type: terms, date_histogram, avg, sum, min, max, percentiles",
                             "enum": ["terms", "date_histogram", "avg", "sum", "min", "max", "percentiles"]},
                "query": {"type": "string", "description": "Optional filter query (Lucene syntax)"},
                "interval": {"type": "string", "description": "For date_histogram: 1h, 1d, 1w"},
                "size": {"type": "integer", "description": "For terms: number of buckets (default 10)"},
            },
            "required": ["index", "field", "agg_type"],
        },
    },
    {
        "name": "get_indices",
        "description": "List available OpenSearch indices with document counts and storage sizes.",
        "inputSchema": {"type": "object", "properties": {}},
    },
    {
        "name": "get_mappings",
        "description": "Get field mappings for an index — shows all fields and their types.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "index": {"type": "string", "description": "Index name or pattern"},
            },
            "required": ["index"],
        },
    },
    {
        "name": "create_index",
        "description": "Create a new OpenSearch index with optional mappings. Requires write scope.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "index": {"type": "string", "description": "Index name (e.g. 'my-logs-2025')"},
                "mappings": {"type": "object", "description": "Optional field mappings"},
                "shards": {"type": "integer", "description": "Number of primary shards (default 1)"},
            },
            "required": ["index"],
        },
    },
    {
        "name": "delete_docs",
        "description": "Delete documents matching a query from an index. Requires write scope.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "index": {"type": "string", "description": "Index name"},
                "query": {"type": "string", "description": "Lucene query matching docs to delete"},
            },
            "required": ["index", "query"],
        },
    },
    {
        "name": "get_cluster_health",
        "description": "Get OpenSearch cluster health status, node count, and shard info.",
        "inputSchema": {"type": "object", "properties": {}},
    },
]


# ── Tool Handlers ──────────────────────────────────────────────────────────────

def handle_tool(name: str, args: dict) -> str:
    """Execute a tool call and return JSON string result."""
    try:
        if name == "search_logs":
            return _search_logs(args)
        if name == "aggregate":
            return _aggregate(args)
        if name == "get_indices":
            return _get_indices()
        if name == "get_mappings":
            return _get_mappings(args)
        if name == "create_index":
            return _create_index(args)
        if name == "delete_docs":
            return _delete_docs(args)
        if name == "get_cluster_health":
            return _get_cluster_health()
        return json.dumps({"error": f"Unknown tool: {name}"})
    except httpx.HTTPStatusError as e:
        return json.dumps({"error": f"HTTP {e.response.status_code}", "detail": e.response.text[:200]})
    except Exception as e:
        return json.dumps({"error": str(e)})


def _search_logs(args):
    index = args["index"]
    query = args["query"]
    size = args.get("size", 10)
    result = _os_request("POST", f"/{index}/_search", {
        "query": {"query_string": {"query": query}},
        "size": size,
    })
    hits = result.get("hits", {})
    total = hits.get("total", {}).get("value", 0)
    docs = [h.get("_source", {}) for h in hits.get("hits", [])]
    return json.dumps({"total": total, "results": docs}, indent=2, default=str)


def _aggregate(args):
    index = args["index"]
    field = args["field"]
    agg_type = args["agg_type"]
    body = {"size": 0, "aggs": {}}

    if agg_type == "terms":
        body["aggs"]["result"] = {"terms": {"field": field, "size": args.get("size", 10)}}
    elif agg_type == "date_histogram":
        body["aggs"]["result"] = {"date_histogram": {
            "field": field, "fixed_interval": args.get("interval", "1h")}}
    elif agg_type in ("avg", "sum", "min", "max"):
        body["aggs"]["result"] = {agg_type: {"field": field}}
    elif agg_type == "percentiles":
        body["aggs"]["result"] = {"percentiles": {"field": field}}

    if args.get("query"):
        body["query"] = {"query_string": {"query": args["query"]}}

    result = _os_request("POST", f"/{index}/_search", body)
    aggs = result.get("aggregations", {}).get("result", {})
    return json.dumps(aggs, indent=2, default=str)


def _get_indices():
    result = _os_request("GET", "/_cat/indices?format=json")
    indices = [{"index": i.get("index"), "docs": i.get("docs.count"),
                "size": i.get("store.size"), "health": i.get("health")}
               for i in result if not i.get("index", "").startswith(".")]
    return json.dumps(indices, indent=2)


def _get_mappings(args):
    index = args["index"]
    result = _os_request("GET", f"/{index}/_mapping")
    # Flatten to {index: {field: type}}
    flat = {}
    for idx, data in result.items():
        props = data.get("mappings", {}).get("properties", {})
        flat[idx] = {k: v.get("type", "object") for k, v in props.items()}
    return json.dumps(flat, indent=2)


def _create_index(args):
    index = args["index"]
    body = {"settings": {"number_of_shards": args.get("shards", 1), "number_of_replicas": 1}}
    if args.get("mappings"):
        body["mappings"] = {"properties": args["mappings"]}
    result = _os_request("PUT", f"/{index}", body)
    return json.dumps(result, indent=2)


def _delete_docs(args):
    index = args["index"]
    query = args["query"]
    result = _os_request("POST", f"/{index}/_delete_by_query", {
        "query": {"query_string": {"query": query}},
    })
    return json.dumps({"deleted": result.get("deleted", 0), "total": result.get("total", 0)}, indent=2)


def _get_cluster_health():
    result = _os_request("GET", "/_cluster/health")
    return json.dumps(result, indent=2)


# ── MCP Server ─────────────────────────────────────────────────────────────────

try:
    from mcp.server import Server
    from mcp.server.stdio import stdio_server
    from mcp.types import Tool, TextContent
    HAS_MCP = True
except ImportError:
    HAS_MCP = False


async def run_mcp_server():
    """Run as MCP server over stdio."""
    server = Server("opensearch-oauth4os")

    @server.list_tools()
    async def list_tools():
        return [Tool(**t) for t in TOOLS]

    @server.call_tool()
    async def call_tool(name: str, arguments: dict) -> list:
        result = handle_tool(name, arguments)
        return [TextContent(type="text", text=result)]

    async with stdio_server() as (read, write):
        await server.run(read, write, server.create_initialization_options())


def main_standalone():
    """Run tools directly from CLI for testing."""
    if len(sys.argv) < 2:
        print("Usage: python server.py <tool> [args_json]")
        print(f"Tools: {', '.join(t['name'] for t in TOOLS)}")
        print(f"\nConfigured: {OAUTH4OS_URL} as {CLIENT_ID} scope={SCOPE}")
        return
    tool = sys.argv[1]
    args = json.loads(sys.argv[2]) if len(sys.argv) > 2 else {}
    print(handle_tool(tool, args))


if __name__ == "__main__":
    if "--stdio" in sys.argv or (HAS_MCP and len(sys.argv) < 2):
        import asyncio
        asyncio.run(run_mcp_server())
    else:
        main_standalone()
