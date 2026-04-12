# ADR-004: Multi-Tenancy Keyed by OIDC Issuer

## Status
Accepted

## Context
Multiple teams/organizations need to share one proxy with different authorization rules.

## Decision
Key tenants by OIDC issuer URL. Each issuer gets its own scope mappings and Cedar policies, with global fallback.

## Rationale
- **Natural boundary** — each IdP (Keycloak realm, Dex instance, Okta org) is already a trust boundary
- **Zero client changes** — tenant is derived from the JWT `iss` claim, not a header or parameter
- **Composable** — global policies apply to all tenants, tenant-specific policies override

## Consequences
- One OIDC provider = one tenant (can't have sub-tenants within a provider)
- Issuer URL must be stable (changing it breaks tenant mapping)
- Config grows linearly with tenant count
