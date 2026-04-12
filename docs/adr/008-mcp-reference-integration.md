# ADR-008: MCP Server as Reference AI Integration

## Status
Accepted

## Context
AI agents (Claude, LangChain, custom) need to query OpenSearch. We needed a reference integration pattern.

## Decision
Build a Model Context Protocol (MCP) server as the reference example, with 7 OpenSearch tools.

## Rationale
- **MCP is the emerging standard** — adopted by Claude, Cursor, and other AI tools
- **Demonstrates the value prop** — scoped, auditable, revocable AI access to OpenSearch
- **Dual mode** — works as MCP server (stdio) and standalone CLI for testing
- **Python** — most AI/ML tooling is Python, lowering the barrier

## Consequences
- Python dependency for the example (not Go)
- MCP spec is still evolving — may need updates
- Only covers read/write/admin patterns (not streaming or subscriptions)
