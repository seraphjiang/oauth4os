# Security Scan — oauth4os

Scan date: 2026-04-12
Scanner: index-eng (manual review)

## Methodology

Manual code review of all HTTP handlers, URL parsing, header handling, and external fetches. Checked for OWASP Top 10 patterns: path traversal, header injection, SSRF, open redirect, injection.

## Findings

### 🔴 HIGH: Open Redirect in PKCE Flow

**File**: `internal/pkce/handler.go`
**Line**: `http.Redirect(w, r, fmt.Sprintf("%s?code=%s", redirectURI, code), http.StatusFound)`

The `redirect_uri` parameter is taken directly from the query string with no validation. An attacker can set `redirect_uri=https://evil.com/steal` and the proxy will redirect the authorization code there.

**Fix**: Validate `redirect_uri` against a pre-registered allowlist per client. At minimum, reject URIs with different schemes/hosts than configured.

**Status**: ⚠️ Needs fix

---

### 🟡 MEDIUM: SSRF via JWKS/OIDC Discovery

**File**: `internal/jwt/validator.go`
**Lines**: 196, 227

The JWKS URI and OIDC discovery URL are derived from the `issuer` field in config. If an attacker can influence the issuer claim in a token (before signature verification), the validator fetches from an attacker-controlled URL.

However, the current flow mitigates this:
1. `findProvider()` only matches issuers from the config file
2. Unknown issuers are rejected before any fetch

**Risk**: Low in current implementation. Would become high if dynamic provider registration is added without URL validation.

**Recommendation**: Add URL scheme validation (https only) and block private IP ranges in JWKS fetches.

**Status**: ✅ Acceptable (config-only providers)

---

### 🟡 MEDIUM: Header Injection via JWT Claims

**File**: `cmd/proxy/main.go:276-278`

```go
r.Header.Set("X-Proxy-User", claims.ClientID)
r.Header.Set("X-Proxy-Roles", strings.Join(roles, ","))
r.Header.Set("X-Proxy-Scopes", strings.Join(claims.Scopes, ","))
```

If `ClientID` or scope values contain newline characters (`\r\n`), this could inject additional HTTP headers. Go's `Header.Set` does sanitize CRLF in values (since Go 1.20), so this is mitigated by the runtime.

**Risk**: Low (Go runtime sanitizes). Would be high in other languages.

**Recommendation**: Explicitly validate ClientID and scope values contain no control characters.

**Status**: ✅ Mitigated by Go runtime

---

### 🟢 LOW: Path Traversal in extractIndex

**File**: `cmd/proxy/main.go`

```go
func extractIndex(path string) string {
    path = strings.TrimPrefix(path, "/")
    if idx := strings.IndexByte(path, '/'); idx > 0 {
        return path[:idx]
    }
    return path
}
```

The extracted index name is used only for Cedar policy evaluation, not for filesystem access. Path traversal sequences like `../` would be passed to OpenSearch, which handles its own path validation.

**Risk**: None — no filesystem access, OpenSearch validates index names.

**Status**: ✅ Safe

---

### 🟢 LOW: No Request Body Size Limit

**File**: `cmd/proxy/main.go`

The proxy forwards request bodies without size limits. A malicious client could send a very large body to exhaust memory.

**Recommendation**: Add `http.MaxBytesReader` for token endpoints. Proxy passthrough can rely on OpenSearch's own limits.

**Status**: ✅ Acceptable for MVP

---

### ✅ PASS: Error Response Leakage

All error responses use generic OAuth 2.0 error codes. No internal paths, stack traces, or upstream URLs exposed. (See docs/error-audit.md for full audit.)

### ✅ PASS: Sensitive Header Forwarding

`Authorization` and `Cookie` headers are stripped before forwarding to upstream on both authenticated and unauthenticated paths.

### ✅ PASS: X-Proxy-* Header Impersonation

On unauthenticated requests, `X-Proxy-User`, `X-Proxy-Roles`, and `X-Proxy-Scopes` are explicitly deleted, preventing clients from injecting trust headers.

## Summary

| Finding | Severity | Status |
|---|---|---|
| Open redirect in PKCE | 🔴 High | Needs fix |
| SSRF via JWKS fetch | 🟡 Medium | Acceptable (config-only) |
| Header injection via claims | 🟡 Medium | Mitigated by Go |
| Path traversal in extractIndex | 🟢 Low | Safe |
| No request body size limit | 🟢 Low | Acceptable for MVP |
| Error response leakage | ✅ Pass | Clean |
| Sensitive header forwarding | ✅ Pass | Clean |
| Header impersonation | ✅ Pass | Clean |

**Action required**: Fix open redirect in PKCE flow before production use.
