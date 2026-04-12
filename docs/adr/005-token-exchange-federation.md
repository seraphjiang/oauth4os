# ADR-005: RFC 8693 Token Exchange as Federation Primitive

## Status
Accepted

## Context
External users (AI agents, CI pipelines, partner services) need OpenSearch access without sharing cluster credentials.

## Decision
Implement RFC 8693 token exchange. Clients present an external JWT (subject_token) and receive a scoped oauth4os token.

## Rationale
- **No credential sharing** — external clients never see OpenSearch passwords or internal tokens
- **Downscoping** — exchange can request fewer scopes than the original token grants
- **Auditable** — exchanged tokens are tracked with `subject@issuer` identity
- **Standard** — RFC 8693 is widely supported by IdPs and token services

## Alternatives Considered
- Direct OIDC passthrough: no scope control, no revocation
- API key issuance: manual process, no federation
- Custom token format: non-standard, harder to integrate

## Consequences
- Two-hop auth flow (IdP → proxy → OpenSearch) adds latency (~5ms)
- Must trust external IdP's JWKS endpoint
- Exchanged tokens are proxy-managed (in-memory for MVP)
