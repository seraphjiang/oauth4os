# ADR-007: Scope Format — action:index-pattern

## Status
Accepted

## Context
Need a scope naming convention that maps cleanly to OpenSearch index permissions.

## Decision
Use `action:index-pattern` format: `read:logs-*`, `write:dashboards`, `admin`.

## Rationale
- **Intuitive** — `read:logs-*` clearly means "read access to log indices"
- **Granular** — action (read/write/admin) × index pattern gives fine-grained control
- **Wildcard support** — `logs-*` matches OpenSearch index patterns natively
- **Familiar** — similar to GitHub's `repo:read` or GCP's `storage.objects.get`

## Consequences
- Scope strings are proxy-specific (not standard OAuth scopes)
- Must document the format for client developers
- Wildcard matching adds complexity to scope validation
