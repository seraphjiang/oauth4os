# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in oauth4os, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

### How to report

Email: **security@oauth4os.dev** (or open a [private security advisory](https://github.com/seraphjiang/oauth4os/security/advisories/new) on GitHub)

Include:
- Description of the vulnerability
- Steps to reproduce
- Affected versions
- Impact assessment (if known)

### Response timeline

| Step | Timeline |
|------|----------|
| Acknowledgment | Within 48 hours |
| Initial assessment | Within 5 business days |
| Fix development | Within 14 days for critical, 30 days for others |
| Public disclosure | After fix is released, coordinated with reporter |

### What qualifies

- Authentication bypass
- Authorization bypass (scope escalation, Cedar policy bypass)
- Token leakage or theft
- Cryptographic weaknesses (JWT validation, PKCE, SigV4)
- Injection attacks (header injection, path traversal)
- Denial of service (resource exhaustion, crash)
- Information disclosure (audit logs, config, credentials)

### What does not qualify

- Issues requiring physical access to the server
- Social engineering
- Denial of service via rate limiting (this is expected behavior)
- Vulnerabilities in dependencies (report to the upstream project)
- Issues in the demo deployment (not production)

## Security Architecture

See [docs/architecture.md](docs/architecture.md) for the full security architecture including:

- 9-stage request middleware chain
- JWT validation with JWKS auto-refresh
- Cedar policy evaluation (deny-overrides)
- PKCE authorization with consent screen
- Refresh token rotation with reuse detection
- SigV4 signing for AOSS
- Mutual TLS client authentication
- Per-client IP allowlist/denylist
- Structured audit logging

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.4.x | ✅ Current |
| 0.3.x | ✅ Security fixes |
| 0.2.x | ❌ End of life |
| 0.1.x | ❌ End of life |

## Dependencies

oauth4os uses only 2 external Go modules to minimize supply chain risk:

| Module | Purpose |
|--------|---------|
| `github.com/golang-jwt/jwt/v5` | JWT parsing and validation |
| `gopkg.in/yaml.v3` | YAML config parsing |

All other functionality uses Go's standard library.
