# ADR-002: Cedar Policies Over Simple RBAC

## Status
Accepted

## Context
We needed authorization beyond scope-to-role mapping. Specific requirement: "permit all authenticated requests, but deny access to `.opendistro_security` index."

## Decision
Implement a Cedar-style policy engine with permit/forbid effects and deny-overrides.

## Rationale
- **Deny-overrides** — "permit all, forbid X" is a common pattern that pure RBAC can't express
- **Composable** — policies stack per-tenant without conflicts
- **Auditable** — every decision includes the matching policy ID and reason
- **Extensible** — can add principal/action/resource matching without changing the evaluation model

## Alternatives Considered
- Simple RBAC: insufficient for deny patterns
- OPA/Rego: too heavy a dependency for a proxy
- Casbin: Go library but adds external dependency

## Consequences
- Custom Cedar implementation (~150 lines) rather than official SDK (no Go binding exists)
- Must document policy syntax for users
- Policy evaluation adds ~microseconds per request (benchmarked)
