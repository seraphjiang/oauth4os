# ADR-010: YAML Config with Runtime Admin API

## Status
Accepted

## Context
Proxy configuration (providers, scopes, Cedar policies, rate limits) needs to be manageable. Options: YAML-only, database-backed, or hybrid.

## Decision
YAML file for initial bootstrap + REST Admin API for runtime changes.

## Rationale
- **YAML for GitOps** — config is version-controlled, reviewable, declarative
- **Admin API for operations** — add a provider or Cedar policy without restarting
- **Backup/restore** — `GET /admin/backup` exports full config as JSON, `POST /admin/restore` imports it
- **No database dependency** — config lives in memory, bootstrapped from YAML

## Consequences
- Runtime changes are lost on restart (unless exported via backup)
- No config change history (mitigated by audit logging)
- Must secure Admin API endpoints (currently no auth — production should require admin scope)
