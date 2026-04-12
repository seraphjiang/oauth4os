# ADR-009: PKCE for Browser Clients Over Implicit Flow

**Status**: Accepted
**Date**: 2026-04-12

## Context

Browser-based clients (OSD plugin, SPAs) need to authenticate. Options: implicit flow, PKCE, backend-for-frontend.

## Decision

Support PKCE (Proof Key for Code Exchange) authorization code flow for browser clients.

## Rationale

- OAuth 2.1 deprecates implicit flow — PKCE is the recommended replacement
- PKCE prevents authorization code interception attacks
- No client secret needed in the browser — only code_verifier/code_challenge
- Backend-for-frontend rejected for MVP: adds a server component, increases complexity

## Consequences

- Requires S256 challenge method support (implemented)
- Authorization codes are short-lived (5 minutes) and single-use
- Open redirect risk mitigated by registered redirect_uri allowlist (fixed in 1f43749)
