# ADR-001: Reverse Proxy, Not OpenSearch Plugin

## Status
Accepted

## Context
We needed to add OAuth 2.0 token management to OpenSearch. Two approaches: build an OpenSearch Security Plugin extension, or put a reverse proxy in front.

## Decision
Build a standalone reverse proxy that sits in front of OpenSearch.

## Rationale
- **Zero OpenSearch changes** — works with any OpenSearch version, no plugin compatibility issues
- **Language freedom** — Go is better suited for a high-performance proxy than Java
- **Independent deployment** — upgrade the proxy without touching the cluster
- **Broader applicability** — same proxy works for Engine and Dashboards
- **Simpler testing** — test the proxy in isolation without a running cluster

## Consequences
- Extra network hop (mitigated by connection pooling — measured <1ms overhead)
- Must configure OpenSearch to trust proxy headers (`X-Proxy-User`, `X-Proxy-Roles`)
- Cannot intercept OpenSearch-internal operations (cluster management, replication)
