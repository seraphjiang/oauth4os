# ADR-009: Structured JSON Logging Over Text

## Status
Accepted

## Context
The original audit logger wrote text lines (`[timestamp] client=X scopes=[Y] GET /path`). Need machine-parseable logs for production.

## Decision
Replace text audit logs with structured JSON lines. One JSON object per line.

## Rationale
- **Machine-parseable** — pipe to CloudWatch, Datadog, ELK, Splunk without custom parsers
- **Queryable** — filter by `client_id`, `method`, `status`, `duration_ms`
- **Extensible** — add fields (request_id, error) without breaking parsers
- **Container-friendly** — stdout JSON is the standard for Docker/Kubernetes logging

## Consequences
- Slightly larger log volume (~2x vs text)
- Human readability reduced (mitigated by `jq` for local dev)
- Old text Auditor kept for backward compatibility
