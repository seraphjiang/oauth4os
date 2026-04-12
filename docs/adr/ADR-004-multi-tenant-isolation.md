# ADR-004: Multi-Tenant Isolation by OIDC Issuer

**Status**: Accepted
**Date**: 2026-04-12

## Context

Multiple teams/orgs need isolated access to the same OpenSearch cluster. Options: separate clusters, index-level isolation, proxy-level isolation.

## Decision

Isolate tenants by OIDC issuer. Each issuer gets its own scope mappings and Cedar policies.

## Rationale

- OIDC issuer is a natural tenant boundary — each org has its own IdP
- No OpenSearch changes needed — isolation enforced at proxy layer
- Cedar policies per tenant enable fine-grained index-level access control
- Separate clusters rejected: expensive, operational overhead
- OpenSearch Security Plugin tenancy rejected: complex, requires plugin configuration

## Consequences

- Tenant config grows with number of issuers — manageable via Admin API
- Cross-tenant queries not supported (by design — security boundary)
