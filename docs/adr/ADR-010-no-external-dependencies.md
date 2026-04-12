# ADR-010: Minimal External Dependencies

**Status**: Accepted
**Date**: 2026-04-12

## Context

Go projects can pull in many transitive dependencies. Need to balance functionality vs supply chain risk.

## Decision

Minimize external dependencies. Use stdlib where possible. Only two external deps: `golang-jwt/jwt` (JWT parsing) and `gopkg.in/yaml.v3` (config).

## Rationale

- Smaller attack surface — fewer deps = fewer CVEs to track
- Faster builds, smaller binary (~10MB)
- Go stdlib covers HTTP server, reverse proxy, TLS, crypto, JSON — no framework needed
- JWT parsing is complex enough to justify a library (RSA verification, claims parsing)
- YAML parsing not in stdlib — yaml.v3 is the de facto standard

## Consequences

- Some features require more code (e.g., Cedar parser is hand-written vs using a library)
- Rate limiter, audit logger, PKCE all implemented from scratch — more code to maintain
- Trade-off is worth it for a security-critical proxy
