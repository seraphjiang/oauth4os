# Troubleshooting

Common issues and fixes for oauth4os.

---

## SigV4 / AOSS

### AOSS returns 403 Forbidden

**Symptom**: Proxy forwards request to AOSS, gets `403 {"message":"User: arn:aws:sts::ACCOUNT:assumed-role/... is not authorized"}`.

**Causes and fixes**:

1. **Missing data access policy** — AOSS requires a data access policy (separate from IAM). Create one in the AOSS console granting your IAM role access to the collection and its indices:
   ```json
   [{"Rules":[
     {"Resource":["collection/your-collection"],"Permission":["aoss:*"],"ResourceType":"collection"},
     {"Resource":["index/your-collection/*"],"Permission":["aoss:*"],"ResourceType":"index"}
   ],"Principal":["arn:aws:iam::ACCOUNT:role/oauth4os-role"]}]
   ```

2. **Wrong IAM role** — Verify which role the proxy is using:
   ```bash
   # On the proxy host / container
   curl -sf http://169.254.169.254/latest/meta-data/iam/security-credentials/
   ```
   For App Runner, check the instance role ARN. For ECS, check the task role (not execution role).

3. **IAM policy missing `aoss:APIAccessAll`** — The IAM policy needs:
   ```json
   {"Effect":"Allow","Action":"aoss:APIAccessAll","Resource":"arn:aws:aoss:REGION:ACCOUNT:collection/COLLECTION_ID"}
   ```

### AOSS returns 404 Not Found

**Symptom**: `{"status":404}` for all requests to AOSS.

**Causes and fixes**:

1. **Wrong Host header** — SigV4 requires the `Host` header to match the AOSS endpoint. Verify your config:
   ```yaml
   upstream:
     engine: https://abc123.us-west-2.aoss.amazonaws.com  # no trailing slash
   ```

2. **Index doesn't exist** — AOSS doesn't auto-create indices. Create it first:
   ```bash
   curl -X PUT https://proxy:8443/my-index \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"settings":{"number_of_shards":1}}'
   ```

3. **SigV4 region mismatch** — The signing region must match the collection's region:
   ```yaml
   upstream:
     sigv4:
       region: us-west-2   # must match AOSS collection region
       service: aoss
   ```

### SigV4 signature mismatch

**Symptom**: `The request signature we calculated does not match the signature you provided`.

**Causes**:
- Clock skew >5 minutes between proxy and AWS. Fix: `ntpdate pool.ntp.org` or ensure NTP is running in the container.
- Request body changed after signing. This shouldn't happen with the proxy — file a bug if it does.
- Stale credentials. The proxy refreshes credentials automatically, but if using static keys, check they haven't been rotated.

---

## Authentication

### Token expired (401)

**Symptom**: `{"error":"invalid_token","error_description":"token expired"}`.

**Fix**: Get a new token or use a refresh token:
```bash
# New token
curl -X POST https://proxy:8443/oauth/token \
  -d "grant_type=client_credentials&client_id=ID&client_secret=SECRET&scope=read:logs-*"

# Refresh
curl -X POST https://proxy:8443/oauth/token \
  -d "grant_type=refresh_token&refresh_token=RT-xxx"
```

Default token TTL is 3600 seconds (1 hour). Refresh tokens last 7 days.

### Invalid JWT signature (401)

**Symptom**: `{"error":"invalid_token","error_description":"signature validation failed"}`.

**Causes**:
1. **JWKS not loaded** — Check `/health/deep` to see if JWKS is cached. The proxy fetches JWKS on first request and refreshes periodically.
2. **Wrong issuer** — The token's `iss` claim must match a configured provider's `issuer` URL exactly (including trailing slash or lack thereof).
3. **Key rotation** — If the IdP rotated keys, the proxy will pick up new keys on the next JWKS refresh (default: 5 minutes). Force refresh by restarting the proxy.

### Token introspection returns inactive

**Symptom**: `POST /oauth/introspect` returns `{"active":false}`.

**Causes**: Token was revoked, expired, or was never issued by this proxy. Only tokens issued by oauth4os can be introspected — external IdP tokens must be exchanged first via token exchange (RFC 8693).

---

## PKCE / Consent

### Redirect URI mismatch

**Symptom**: `{"error":"invalid_request","error_description":"redirect_uri not registered for client"}`.

**Fix**: The `redirect_uri` in the authorize request must exactly match a URI registered for the client. Register it:
```bash
curl -X POST https://proxy:8443/oauth/register \
  -H "Content-Type: application/json" \
  -d '{"client_name":"my-app","redirect_uris":["http://localhost:3000/callback"],"scope":"read:logs-*"}'
```

Common mistakes:
- `http` vs `https`
- Trailing slash mismatch (`/callback` vs `/callback/`)
- Port mismatch (`localhost:3000` vs `localhost:8080`)

### Consent expired

**Symptom**: `{"error":"invalid_request","error_description":"consent expired or not found"}`.

**Fix**: Consent screens expire after 10 minutes. Start the flow again from `/oauth/authorize`. Don't bookmark or share consent URLs.

### Code verifier failed

**Symptom**: `{"error":"invalid_grant","error_description":"code_verifier failed verification"}`.

**Fix**: The SHA256 hash of the `code_verifier` must match the `code_challenge` sent in the authorize request. Ensure:
- You're using S256 method (not plain)
- The verifier is the original random string, not the challenge
- Base64 URL encoding without padding (`=`)

---

## Rate Limiting

### 429 Too Many Requests

**Symptom**: `HTTP 429` with `Retry-After` header.

**Fix**: Wait for the duration in the `Retry-After` header, then retry. To increase limits, update config:
```yaml
rate_limits:
  "read:logs-*": 600    # requests per minute
  "admin": 60
```

Or via the Admin API:
```bash
curl -X PUT https://proxy:8443/admin/rate-limits \
  -H "Content-Type: application/json" \
  -d '{"read:logs-*": 1200}'
```

### Rate limit applies to wrong scope

Each client is rate-limited by their most restrictive scope. If a token has both `read:logs-*` (600 RPM) and `admin` (60 RPM), the `admin` limit applies to admin operations and `read` limit to read operations. They are tracked independently.

---

## CORS

### Browser gets CORS error

**Symptom**: `Access to fetch at 'https://proxy:8443/...' from origin 'http://localhost:3000' has been blocked by CORS policy`.

**Fix**: The proxy doesn't set CORS headers by default. For browser clients, put a CORS-aware reverse proxy (nginx, CloudFront, ALB) in front, or add CORS headers in your deployment:

**nginx example**:
```nginx
location / {
    proxy_pass https://oauth4os:8443;
    add_header Access-Control-Allow-Origin $http_origin always;
    add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Authorization, Content-Type" always;
    if ($request_method = OPTIONS) { return 204; }
}
```

The `/demo` app works without CORS issues because it's served from the same origin as the proxy.

---

## Docker Build

### `go mod download` fails with TLS errors

**Symptom**: `tls: failed to verify certificate: x509: certificate is valid for chalupa-dns-sinkhole.corp.amazon.com`.

**Fix**: Corporate proxy intercepting TLS. Build with direct module fetching:
```bash
docker build --build-arg GOPROXY=direct --build-arg GONOSUMDB='*' -t oauth4os .
```

Or in the Dockerfile (already set):
```dockerfile
ENV GOPROXY=direct GONOSUMDB=* GOFLAGS=-insecure
```

### Build fails with `buildvcs` error

**Symptom**: `error obtaining VCS status: exit status 128`.

**Fix**: Add `-buildvcs=false` to the build command. The Dockerfile already does this:
```dockerfile
RUN CGO_ENABLED=0 go build -buildvcs=false -o oauth4os ./cmd/proxy
```

### Image too large

The multi-stage Dockerfile produces a ~15MB image (Alpine + static binary). If you see a larger image, ensure you're using the multi-stage build and not copying the entire build context.

---

## Cedar Policies

### Cedar denies a request that should be allowed

**Symptom**: `403 Forbidden` even though the token has the right scopes.

**Debug**:
1. Check which policies are loaded:
   ```bash
   curl https://proxy:8443/admin/policies
   ```
2. Cedar uses deny-overrides — any `forbid` rule matching the request will deny it, even if a `permit` rule also matches.
3. Check tenant-specific policies. If the token's issuer matches a tenant, that tenant's policies are evaluated in addition to global policies.

### No Cedar policies but still getting 403

If no Cedar policies are configured, the proxy defaults to permit-all (Cedar is optional). A 403 without Cedar means the scope mapping didn't produce any backend roles. Check:
```bash
curl https://proxy:8443/admin/scope-mappings
```

---

## Connectivity

### Proxy can't reach OpenSearch

**Symptom**: `/health/deep` shows upstream as unhealthy.

**Checks**:
1. Verify the upstream URL is reachable from the proxy container:
   ```bash
   docker exec oauth4os wget -qO- https://opensearch:9200 --no-check-certificate
   ```
2. For AOSS, ensure the security group allows outbound HTTPS (443) to the AOSS endpoint.
3. For self-signed certs, set `tls.insecure_skip_verify: true` in config (development only).

### Proxy starts but no requests get through

**Checks**:
1. Verify the listen port matches your load balancer / port mapping: `listen: :8443`
2. Check logs: `docker logs oauth4os`
3. Test health: `curl http://localhost:8443/health`
4. If using TLS, ensure the client is connecting via HTTPS.

---

## Getting Help

1. Check proxy logs: `docker logs oauth4os` — all errors are logged as structured JSON
2. Check deep health: `curl https://proxy:8443/health/deep`
3. Check audit log: `curl https://proxy:8443/admin/audit?limit=10`
4. File an issue: https://github.com/seraphjiang/oauth4os/issues
