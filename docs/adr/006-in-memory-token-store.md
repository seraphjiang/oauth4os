# ADR-006: In-Memory Token Store for MVP

## Status
Accepted (revisit for production)

## Context
Tokens need to be stored for validation, introspection, and revocation. Options: in-memory, DynamoDB, Redis, OpenSearch itself.

## Decision
Store tokens in-memory with `sync.RWMutex` for the MVP.

## Rationale
- **Simplicity** — no external dependencies for the POC
- **Performance** — O(1) lookup, no network round-trip
- **Sufficient for demo** — single-instance proxy doesn't need distributed state

## Known Limitations
- Tokens lost on proxy restart
- No horizontal scaling (each instance has its own token set)
- No persistence for audit/compliance

## Migration Path
Add a `TokenStore` interface with `InMemory`, `DynamoDB`, and `Redis` implementations. The Manager already isolates storage behind `createToken`/`Lookup` methods.
