#!/usr/bin/env python3
"""MCP Server — AI agent access to OpenSearch via oauth4os.

Authenticates with client_credentials, exposes search/indices/health
tools over the Model Context Protocol (stdio transport).

Requires: pip install mcp httpx
"""

import json
import os
import sys
import time
from typing import Any

import httpx

# ── oauth4os Token Client ──────────────────────────────────────────────────────

OAUTH4OS_URL = os.environ.get("OAUTH4OS_URL", "http://localhost:8443")
CLIENT_ID = os.environ.get("OAUTH4OS_CLIENT_ID", "mcp-agent")
CLIENT_SECRET = os.environ.get("OAUTH4OS_CLIENT_SECRET", "secret")
SCOPE = os.environ.get("OAUTH4OS_SCOPE", "read:logs-*")

_token_cache: dict = {}


def _get_token() -> str:
    """Get a valid access token, refreshing if expired."""
    if _token_cache.get("token") and _token_cache.get("expires_at", 0) > time.time() + 60:
        return _token_cache["token"]

    resp = httpx.post(
        f"{OAUTH4OS_URL}/oauth/token",
        data={
            "grant_type": "client_credentials",
            "client_id": CLIENT_ID,
            "client_secret": CLIENT_SECRET,
            "scope": SCOPE,
        },
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
        method,
        f"{OAUTH4OS_URL}{path}",
        headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
        content=json.dumps(body) if body else None,
        timeout=30,
    )
    resp.raise_for_status()
    return resp.json()


# ── MCP Server ─────────────────────────────────────────────────────────────────

try:
    from mcp.server import Server
    from mcp.server.stdio import stdio_server
    from mcp.types import Tool, TextContent
    HAS_MCP = True
except ImportError:
    HAS_MCP = False

TOOLS = [
    {
        "name": "search_logs",
        "description": "Search an OpenSearch index. Returns matching documents.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "index": {"type": "string", "description": "Index pattern (e.g. 'logs-*')"},
                "query": {"type": "string", "description": "Search query string"},
                "size": {"type": "integer", "description": "Max results (default 10)"},
            },
            "required": ["index", "query"],
        },
    },
    {
        "name": "get_indices",
        "description": "List available OpenSearch indices with doc counts and sizes.",
        "inputSchema": {"type": "object", "properties": {}},
    },
    {
        "name": "get_cluster_health",
        "description": "Get OpenSearch cluster health status.",
        "inputSchema": {"type": "object", "properties": {}},
    },
]


def handle_tool(name: str, args: dict) -> str:
    """Execute a tool call and return the result as a string."""
    if name == "search_logs":
        index = args.get("index", "logs-*")
        query = args.get("query", "*")
        size = args.get("size", 10)
        result = _os_request("POST", f"/{index}/_search", {
            "query": {"query_string": {"query": query}},
            "size": size,
        })
        hits = result.get("hits", {})
        total = hits.get("total", {}).get("value", 0)
        docs = [h.get("_source", {}) for h in hits.get("hits", [])]
        return json.dumps({"total": total, "results": docs}, indent=2, default=str)

    if name == "get_indices":
        result = _os_request("GET", "/_cat/indices?format=json")
        indices = [{"index": i.get("index"), "docs": i.get("docs.count"),
                     "size": i.get("store.size")} for i in result if not i.get("index", "").startswith(".")]
        return json.dumps(indices, indent=2)

    if name == "get_cluster_health":
        result = _os_request("GET", "/_cluster/health")
        return json.dumps(result, indent=2)

    return json.dumps({"error": f"Unknown tool: {name}"})


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


# ── Standalone mode (for testing without MCP) ──────────────────────────────────

def main_standalone():
    """Run tools directly from CLI for testing."""
    if len(sys.argv) < 2:
        print("Usage: python server.py <tool> [args_json]")
        print("Tools: search_logs, get_indices, get_cluster_health")
        print(f"\nConfigured: {OAUTH4OS_URL} as {CLIENT_ID} scope={SCOPE}")
        return

    tool = sys.argv[1]
    args = json.loads(sys.argv[2]) if len(sys.argv) > 2 else {}
    print(handle_tool(tool, args))


if __name__ == "__main__":
    if "--stdio" in sys.argv or HAS_MCP:
        import asyncio
        asyncio.run(run_mcp_server())
    else:
        main_standalone()
