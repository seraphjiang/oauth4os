# ADR-008: Cedar Forbid-Overrides-Permit Semantics

**Status**: Accepted
**Date**: 2026-04-12

## Context

When multiple Cedar policies match a request, need to resolve conflicts. Options: first-match, most-specific, forbid-overrides.

## Decision

Forbid-overrides-permit: any matching forbid policy denies the request, regardless of permit policies.

## Rationale

- Matches real Cedar semantics — users familiar with AWS Verified Permissions get expected behavior
- Security-first: explicit denies can never be overridden by permits
- Enables "allow everything except security indices" pattern — the most common OpenSearch use case
- First-match rejected: order-dependent, fragile, hard to reason about
- Most-specific rejected: complex to implement, ambiguous for glob patterns

## Consequences

- Cannot create "exception to a deny" — must restructure policies instead
- Policy ordering doesn't matter (good for maintainability)
