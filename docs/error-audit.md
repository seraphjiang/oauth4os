# Error Path Security Audit — oauth4os

Audit date: 2026-04-12
Auditor: index-eng

## Summary

All error paths reviewed. No internal details leaked to clients. Headers properly sanitized.

## Findings

### ✅ Error Responses (No Leakage)

| Endpoint | Error | Response | Status |
|---|---|---|---|
| Proxy (auth) | Invalid JWT | `{"error":"invalid_token"}` | 401 |
| Proxy (scope) | No matching roles | `{"error":"insufficient_scope"}` | 403 |
| Proxy (Cedar) | Policy denied | `{"error":"forbidden"}` | 403 |
| Proxy (upstream) | Backend down | `{"error":"upstream_error","message":"upstream unavailable"}` | 502 |
| Token (grant) | Bad grant type | `{"error":"unsupported_grant_type"}` | 400 |
| Token (auth) | Bad credentials | `{"error":"invalid_client"}` | 401 |
| Token (scope) | Exceeds allowance | `{"error":"invalid_scope"}` | 400 |
| Token (refresh) | Reuse detected | `{"error":"invalid_grant"}` | 400 |
| Token (lookup) | Not found | `{"error":"not_found"}` | 404 |

All error messages are generic OAuth 2.0 error codes. No stack traces, no internal paths, no upstream URLs.

### ✅ Header Sanitization

| Header | Unauthenticated Path | Authenticated Path |
|---|---|---|
| `Authorization` | Passed through (existing auth) | Stripped before upstream |
| `Cookie` | Stripped | Stripped |
| `X-Proxy-User` | Stripped (prevents impersonation) | Set by proxy |
| `X-Proxy-Roles` | Stripped | Set by proxy |
| `X-Proxy-Scopes` | Stripped | Set by proxy |

### ✅ JWT Validator Internal Errors

The JWT validator returns detailed errors internally (e.g., `"JWKS fetch failed"`, `"unknown issuer: X"`). These are NOT exposed to clients — the proxy handler catches all validator errors and returns generic `{"error":"invalid_token"}`.

### ⚠️ Recommendations (Low Priority)

1. **Rate limit token endpoint**: Already implemented via rate limiter middleware.
2. **Log auth failures server-side**: Consider structured logging of failed auth attempts (client IP, error type) for security monitoring. Currently only audit-logged on success.
3. **Cedar policy ID in deny response**: The `decision.Policy` field is included in the 403 response in the original code but stripped in the hardened version. Confirm this stays stripped — policy IDs could reveal internal naming.

## Conclusion

No action items. Error paths are clean. Headers are properly sanitized. The proxy does not leak internal details.
