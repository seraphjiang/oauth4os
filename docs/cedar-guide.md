# Cedar Policy Guide

oauth4os uses a Cedar-like policy engine for fine-grained authorization. This guide covers policy syntax, evaluation semantics, multi-tenant isolation, and example policies for common OpenSearch access patterns.

---

## How Cedar Fits In

Cedar policies are evaluated after JWT validation and scope mapping. They provide an additional authorization layer — even if a token has the right scopes, a Cedar policy can deny specific actions.

```
JWT Valid? → Scopes Mapped? → Cedar Permits? → Forward to OpenSearch
                                    │
                              Any forbid → 403
                              No permit  → 403
```

Cedar is optional. If no policies are configured, the proxy defaults to permit-all (scope mapping alone controls access).

---

## Policy Structure

Each policy has:

```
<effect>(principal, action, resource)
  when { <conditions> }
  unless { <exceptions> };
```

| Field | Description | Examples |
|-------|-------------|---------|
| `effect` | `permit` or `forbid` | |
| `principal` | Who — client ID or `*` for any | `my-agent`, `*` |
| `action` | HTTP method or `*` | `GET`, `POST`, `PUT`, `DELETE`, `*` |
| `resource` | OpenSearch index pattern or `*` | `logs-*`, `.opendistro_security`, `*` |
| `when` | Conditions that must be true | scope check, IP check |
| `unless` | Exceptions that skip the policy | specific index exclusion |

---

## Evaluation: Deny-Overrides

Cedar uses a deny-overrides model:

1. All policies matching the request are evaluated
2. If **any** `forbid` matches → **DENY** (immediate, no further evaluation)
3. If **at least one** `permit` matches → **ALLOW**
4. If **no** policies match → **DENY** (default deny)

This means a single `forbid` rule always wins, regardless of how many `permit` rules exist. This is the key security property — you can grant broad access and then carve out exceptions.

```
permit(*, GET, *)           ← allows all reads
forbid(*, *, .internal*)    ← but blocks internal indices

Result for GET .internal_config → DENIED (forbid wins)
Result for GET logs-demo       → ALLOWED (permit matches, no forbid)
```

---

## Policy Syntax in config.yaml

Policies are defined in `config.yaml` as strings:

```yaml
# Global policies (apply to all tenants)
cedar_policies:
  - "permit(*, GET, *)"
  - "permit(*, POST, logs-*)"
  - "forbid(*, *, .opendistro_security)"
  - "forbid(*, DELETE, *)"
```

### Shorthand format

```
<effect>(<principal>, <action>, <resource>)
```

- `*` matches anything
- Glob patterns: `logs-*` matches `logs-demo`, `logs-app-a`, etc.
- Exact match: `logs-demo` matches only `logs-demo`

### With conditions

```yaml
cedar_policies:
  - |
    permit(*, GET, logs-*)
      when { principal.scope contains "read:logs-*" }
  - |
    forbid(*, PUT, .internal*)
      unless { principal.sub == "admin-agent" }
```

---

## Policy API

Manage policies at runtime via the Admin API:

```bash
# List all policies
curl https://proxy:8443/admin/policies

# Add a policy
curl -X POST https://proxy:8443/admin/policies \
  -H "Content-Type: application/json" \
  -d '{
    "id": "deny-security-index",
    "effect": "forbid",
    "principal": {"any": true},
    "action": {"any": true},
    "resource": {"pattern": ".opendistro_security"}
  }'

# Remove a policy
curl -X DELETE https://proxy:8443/admin/policies/deny-security-index
```

### Policy JSON format

```json
{
  "id": "unique-policy-id",
  "effect": "permit",
  "principal": {"any": true},
  "action": {"equals": "GET"},
  "resource": {"pattern": "logs-*"},
  "when": [
    {"field": "principal.scope", "op": "contains", "value": "read:logs-*"}
  ],
  "unless": [
    {"field": "resource.index", "op": "==", "value": ".opendistro_security"}
  ]
}
```

### Match types

| Match | JSON | Matches |
|-------|------|---------|
| Any | `{"any": true}` | Everything |
| Exact | `{"equals": "logs-demo"}` | Only `logs-demo` |
| Pattern | `{"pattern": "logs-*"}` | `logs-demo`, `logs-app-a`, etc. |

### Condition operators

| Operator | Description | Example |
|----------|-------------|---------|
| `==` | Exact equality | `principal.sub == "admin"` |
| `!=` | Not equal | `resource.index != ".kibana"` |
| `in` | Value in comma-separated list | `action in "GET,POST"` |
| `contains` | String contains substring | `principal.scope contains "read:"` |

---

## Multi-Tenant Isolation

Each OIDC issuer (tenant) can have its own policy set. Tenant policies are evaluated instead of global policies when the token's issuer matches.

```yaml
tenants:
  "https://keycloak.example.com/realms/team-a":
    cedar_policies:
      - "permit(*, GET, logs-team-a-*)"
      - "permit(*, POST, logs-team-a-*)"
      - "forbid(*, *, logs-team-b-*)"

  "https://keycloak.example.com/realms/team-b":
    cedar_policies:
      - "permit(*, GET, logs-team-b-*)"
      - "forbid(*, *, logs-team-a-*)"

# Global fallback (used when issuer doesn't match any tenant)
cedar_policies:
  - "permit(*, GET, *)"
  - "forbid(*, *, .opendistro_*)"
```

### Evaluation flow

```
Token issuer = "https://keycloak.example.com/realms/team-a"
    → Match tenant "team-a"
    → Evaluate team-a policies ONLY
    → Global policies are NOT evaluated

Token issuer = "https://unknown-idp.example.com"
    → No tenant match
    → Evaluate global policies
```

This provides hard isolation — Team A's tokens can never access Team B's indices, even if the scopes would otherwise allow it.

---

## Example Policies

### 1. Read-only access

Allow all reads, deny all writes:

```yaml
cedar_policies:
  - "permit(*, GET, *)"
  - "permit(*, POST, */_search)"
  - "permit(*, POST, */_count)"
  - "permit(*, POST, */_msearch)"
  - "forbid(*, PUT, *)"
  - "forbid(*, DELETE, *)"
  - "forbid(*, POST, */_bulk)"
```

### 2. Admin with security index protection

Full access except the security index:

```yaml
cedar_policies:
  - "permit(*, *, *)"
  - "forbid(*, *, .opendistro_security)"
  - "forbid(*, *, .plugins-ml-config)"
```

### 3. Per-index access by client

Each agent can only access its own indices:

```yaml
cedar_policies:
  - |
    permit(payment-agent, *, logs-payment-*)
  - |
    permit(auth-agent, *, logs-auth-*)
  - |
    permit(shipping-agent, GET, logs-shipping-*)
  - |
    forbid(*, *, .opendistro_*)
```

### 4. Time-based read-only (business hours)

Use conditions to restrict write access:

```yaml
cedar_policies:
  - "permit(*, GET, *)"
  - |
    permit(*, POST, *)
      when { principal.scope contains "write:" }
  - |
    forbid(*, DELETE, *)
      unless { principal.sub == "admin" }
```

### 5. IP-restricted admin access

Combine Cedar with IP filter for defense in depth:

```yaml
cedar_policies:
  - "permit(*, GET, *)"
  - |
    permit(*, *, *)
      when { principal.scope contains "admin" }
  - "forbid(*, DELETE, *)"

ip_filter:
  clients:
    admin-agent:
      allow: ["10.0.0.0/8"]
```

### 6. Deny specific operations

Block dangerous operations for all clients:

```yaml
cedar_policies:
  - "permit(*, *, *)"
  - "forbid(*, DELETE, *)"
  - "forbid(*, PUT, _cluster/*)"
  - "forbid(*, POST, _reindex)"
  - "forbid(*, POST, */_close)"
  - "forbid(*, *, .opendistro_*)"
```

### 7. Multi-tenant SaaS

Complete isolation between tenants:

```yaml
tenants:
  "https://idp.example.com/tenant-acme":
    cedar_policies:
      - "permit(*, *, acme-*)"
      - "forbid(*, *, *)"

  "https://idp.example.com/tenant-globex":
    cedar_policies:
      - "permit(*, GET, globex-*)"
      - "forbid(*, *, *)"

cedar_policies:
  - "forbid(*, *, *)"
```

Note the global `forbid(*, *, *)` — if a token doesn't match any tenant, everything is denied.

---

## Testing Policies

### Unit test approach

Use the Cedar engine directly in Go tests:

```go
package cedar_test

import (
    "testing"
    "github.com/seraphjiang/oauth4os/internal/cedar"
)

func TestReadOnlyPolicy(t *testing.T) {
    engine := cedar.NewEngine([]cedar.Policy{
        {ID: "allow-read", Effect: cedar.Permit,
            Principal: cedar.Match{Any: true},
            Action:    cedar.Match{Equals: "GET"},
            Resource:  cedar.Match{Any: true}},
        {ID: "deny-write", Effect: cedar.Forbid,
            Principal: cedar.Match{Any: true},
            Action:    cedar.Match{Equals: "PUT"},
            Resource:  cedar.Match{Any: true}},
    })

    // GET should be allowed
    d := engine.Evaluate(cedar.Request{
        Principal: map[string]string{"sub": "agent"},
        Action:    "GET",
        Resource:  map[string]string{"index": "logs-demo"},
    })
    if !d.Allowed {
        t.Error("GET should be allowed")
    }

    // PUT should be denied
    d = engine.Evaluate(cedar.Request{
        Principal: map[string]string{"sub": "agent"},
        Action:    "PUT",
        Resource:  map[string]string{"index": "logs-demo"},
    })
    if d.Allowed {
        t.Error("PUT should be denied")
    }
}
```

### Integration test with curl

```bash
# Get a read-only token
TOKEN=$(curl -sf -X POST https://proxy:8443/oauth/token \
  -d "grant_type=client_credentials&client_id=reader&client_secret=secret&scope=read:logs-*" \
  | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

# Read should work
curl -sf -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $TOKEN" \
  https://proxy:8443/logs-demo/_search
# Expected: 200

# Write should be denied by Cedar
curl -sf -o /dev/null -w "%{http_code}" \
  -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  https://proxy:8443/logs-demo/_doc/test -d '{"test":true}'
# Expected: 403
```

### Debugging policy decisions

Check the audit log to see which policy was evaluated:

```bash
curl "https://proxy:8443/admin/audit?limit=5" | python3 -m json.tool
```

Each audit entry includes:
- `cedar_decision`: `allowed` or `denied`
- `cedar_policy`: ID of the matching policy
- `cedar_reason`: why the decision was made

---

## Best Practices

1. **Start with deny-all, add permits** — safer than permit-all with exceptions:
   ```yaml
   cedar_policies:
     - "permit(*, GET, logs-*)"
     # Everything else is implicitly denied (no matching permit)
   ```

2. **Always protect internal indices** — add a forbid for `.opendistro_*` and `.plugins-*`:
   ```yaml
   - "forbid(*, *, .opendistro_*)"
   - "forbid(*, *, .plugins-*)"
   ```

3. **Use tenant isolation for multi-team** — don't rely on scope naming alone. Cedar tenant policies provide hard boundaries.

4. **Test policies before deploying** — use the Go test approach above or the Admin API to add/remove policies without restart.

5. **Audit regularly** — review `/admin/audit` for denied requests. Unexpected denials may indicate misconfigured policies; unexpected allows may indicate missing forbid rules.

6. **Layer with other controls** — Cedar works alongside scope mapping, rate limiting, and IP filtering. Use all layers for defense in depth.
