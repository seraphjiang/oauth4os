# ADR-003: Self-Issued Tokens for client_credentials

**Status**: Accepted
**Date**: 2026-04-12

## Context

The proxy needs to issue tokens for machine-to-machine (M2M) auth. Options: delegate to external IdP, or self-issue.

## Decision

Self-issue opaque tokens (tok_xxx) for client_credentials grant. External JWT tokens validated via JWKS for federated auth.

## Rationale

- M2M clients (CI/CD, agents, scripts) need tokens without browser-based OIDC flows
- Self-issued tokens are simpler — no external IdP dependency for basic use cases
- Opaque tokens (not JWT) chosen for self-issued: revocation is instant (check in-memory map), no JWT expiry lag
- External JWTs still supported for federated scenarios (Keycloak, Auth0, Okta)

## Consequences

- Token state is in-memory — lost on restart. Acceptable for MVP; persistence (DDB) planned.
- Two token formats: opaque (self-issued) and JWT (external). Validator handles both.
