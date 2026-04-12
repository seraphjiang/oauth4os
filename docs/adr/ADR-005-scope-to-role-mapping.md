# ADR-005: Scope-to-Role Mapping Over Direct Role Assignment

**Status**: Accepted
**Date**: 2026-04-12

## Context

Need to bridge OAuth scopes to OpenSearch backend roles. Options: direct role headers, scope-to-role mapping, dynamic role creation.

## Decision

Map OAuth scopes to OpenSearch backend roles via a configurable mapping table. Inject roles via X-Proxy-Roles header.

## Rationale

- Decouples OAuth semantics from OpenSearch internals — clients request scopes, not roles
- Mapping table is auditable and version-controlled (YAML config)
- Supports many-to-many: one scope can map to multiple roles, one role can be reached by multiple scopes
- Direct role assignment rejected: leaks OpenSearch internals to clients
- Dynamic role creation rejected: requires OpenSearch Security Plugin API access, adds complexity

## Consequences

- Mapping must be maintained as OpenSearch roles change
- Admin API allows runtime updates without restart
