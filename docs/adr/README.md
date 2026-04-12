# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for oauth4os.

| ADR | Title | Status |
|-----|-------|--------|
| [001](001-proxy-not-plugin.md) | Reverse proxy, not OpenSearch plugin | Accepted |
| [002](002-cedar-over-rbac.md) | Cedar policies over simple RBAC | Accepted |
| [003](003-go-stdlib-only.md) | Go stdlib only, no web framework | Accepted |
| [004](004-multi-tenant-by-issuer.md) | Multi-tenancy keyed by OIDC issuer | Accepted |
| [005](005-token-exchange-federation.md) | RFC 8693 token exchange as federation primitive | Accepted |
| [006](006-in-memory-token-store.md) | In-memory token store for MVP | Accepted |
| [007](007-scope-format.md) | Scope format: action:index-pattern | Accepted |
| [008](008-mcp-reference-integration.md) | MCP server as reference AI integration | Accepted |
| [009](009-structured-json-logging.md) | Structured JSON logging over text | Accepted |
| [010](010-config-yaml-plus-admin-api.md) | YAML config with runtime Admin API | Accepted |

### Extended Series

| ADR | Title | Status |
|-----|-------|--------|
| [ADR-001](ADR-001-go-reverse-proxy.md) | Go reverse proxy architecture | Accepted |
| [ADR-002](ADR-002-cedar-policy-engine.md) | Cedar policy engine | Accepted |
| [ADR-003](ADR-003-self-issued-tokens.md) | Self-issued tokens for demo mode | Accepted |
| [ADR-004](ADR-004-multi-tenant-isolation.md) | Multi-tenant isolation | Accepted |
| [ADR-005](ADR-005-scope-to-role-mapping.md) | Scope-to-role mapping | Accepted |
| [ADR-006](ADR-006-function-url-deployment.md) | Function URL deployment | Accepted |
| [ADR-007](ADR-007-yaml-config.md) | YAML configuration | Accepted |
| [ADR-008](ADR-008-forbid-overrides-permit.md) | Forbid overrides permit | Accepted |
| [ADR-009](ADR-009-pkce-for-browsers.md) | PKCE for browser flows | Accepted |
| [ADR-010](ADR-010-no-external-dependencies.md) | Minimal external dependencies | Accepted |
