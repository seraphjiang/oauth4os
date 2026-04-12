# ADR-007: YAML Config with Runtime Admin API

**Status**: Accepted
**Date**: 2026-04-12

## Context

Need to configure providers, scopes, policies. Options: YAML-only, database-backed, environment variables.

## Decision

YAML file for initial config, Admin API for runtime changes. Runtime state is in-memory, seeded from YAML on startup.

## Rationale

- YAML is version-controllable, diff-friendly, familiar to ops teams
- Admin API enables runtime changes without restart (scope mappings, Cedar policies, providers)
- Environment variables rejected for complex config: nested structures don't map well
- Database-backed rejected for MVP: adds infrastructure dependency (DDB/Postgres)

## Consequences

- Runtime changes are lost on restart — must update YAML for persistence
- Backup/restore API planned to bridge the gap
