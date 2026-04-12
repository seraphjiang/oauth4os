# ADR-001: Go Reverse Proxy Architecture

**Status**: Accepted
**Date**: 2026-04-12

## Context

OpenSearch needs an OAuth 2.0 auth layer. Options: sidecar (Envoy/nginx), SDK library, or custom proxy.

## Decision

Build a standalone Go reverse proxy using `net/http/httputil.ReverseProxy`.

## Rationale

- Single binary, zero runtime dependencies — deploys anywhere (Lambda, ECS, K8s, bare metal)
- Go's stdlib reverse proxy is production-grade (used by Caddy, Traefik)
- Sub-millisecond auth overhead — Go's HTTP stack is fast enough that the proxy adds negligible latency
- Sidecar (Envoy) rejected: too complex for the problem, requires Lua/Wasm for custom auth logic
- SDK library rejected: requires changes to every client, not transparent

## Consequences

- Must handle all HTTP edge cases (WebSocket upgrade, chunked transfer, etc.)
- Single point of failure — mitigated by horizontal scaling
- Go-only codebase limits contributions from non-Go developers
