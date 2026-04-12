# ADR-002: Cedar-like Policy Engine

**Status**: Accepted
**Date**: 2026-04-12

## Context

Need fine-grained access control beyond scope-to-role mapping. Options: OPA/Rego, Cedar, custom DSL.

## Decision

Implement a lightweight Cedar-like policy engine in Go. Subset of Cedar syntax: permit/forbid, when/unless conditions, glob matching.

## Rationale

- Cedar's semantics (forbid-overrides-permit) are ideal for security policies
- Full Cedar SDK is Rust-only — embedding via CGO adds complexity and breaks Lambda deployment
- OPA/Rego rejected: powerful but complex syntax, overkill for index-level access control
- Our subset covers 95% of OpenSearch access patterns with ~250 lines of Go
- Familiar to AWS users (Cedar is used by Verified Permissions)

## Consequences

- Not a full Cedar implementation — advanced features (entity hierarchy, IP conditions) not supported
- Must document which Cedar features are/aren't supported
- Migration path to full Cedar if demand grows
