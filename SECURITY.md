# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in oauth4os, please report it responsibly:

1. **Do NOT** open a public GitHub issue
2. Email: security@oauth4os.dev (or use GitHub's private vulnerability reporting)
3. Include: description, steps to reproduce, potential impact

We will acknowledge receipt within 48 hours and provide a fix timeline within 7 days.

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.1.x   | ✅ Current |

## Security Considerations

oauth4os is a security-critical component. Key areas:

- **JWT validation**: Signature verification via JWKS, issuer validation, expiry checks
- **Scope enforcement**: OAuth scopes mapped to OpenSearch backend roles
- **Cedar policies**: Fine-grained access control on index patterns
- **Token management**: Secure generation, revocation, audit trail
- **Proxy**: No credential leakage, request sanitization
