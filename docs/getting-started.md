# Getting Started Tutorial

A hands-on lab guide. You'll register a client, get a token, query OpenSearch through the proxy, and revoke the token — all in about 10 minutes.

## Prerequisites

```bash
docker run -p 8443:8443 jianghuan/oauth4os:latest
```

Verify it's running:

```bash
curl -s http://localhost:8443/health | jq
```

Expected:
```json
{"status": "ok", "version": "1.0.0"}
```

---

## Step 1: Register a Client

Every application that talks to the proxy needs a client identity.

```bash
curl -s -X POST http://localhost:8443/oauth/register \
  -H "Content-Type: application/json" \
  -d '{"client_name": "my-app", "scope": "read:logs-*"}' | jq
```

Expected:
```json
{
  "client_id": "abc123...",
  "client_secret": "secret_xyz...",
  "client_name": "my-app",
  "scope": "read:logs-*"
}
```

Save these — you'll need them in the next step:
```bash
export CLIENT_ID="<client_id from above>"
export CLIENT_SECRET="<client_secret from above>"
```

**What happened**: The proxy created a client record with the scopes you requested. The `client_secret` is shown once — store it securely.

---

## Step 2: Get an Access Token

Use the client credentials flow to get a token:

```bash
curl -s -X POST http://localhost:8443/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=$CLIENT_ID&client_secret=$CLIENT_SECRET&scope=read:logs-*" | jq
```

Expected:
```json
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read:logs-*"
}
```

Save the token:
```bash
export TOKEN="<access_token from above>"
```

**What happened**: The proxy verified your client credentials, checked that the requested scopes are allowed, and issued a signed JWT. The token is self-contained — it carries your identity and permissions.

---

## Step 3: Make an Authenticated Request

Use the token to search the demo index:

```bash
curl -s http://localhost:8443/logs-demo/_search \
  -H "Authorization: Bearer $TOKEN" | jq '.hits.total'
```

Expected:
```json
{"value": 500, "relation": "eq"}
```

Try without a token — you'll get a 401:
```bash
curl -s http://localhost:8443/logs-demo/_search
```

Expected:
```json
{"error": "missing or invalid token"}
```

**What happened**: The proxy validated your JWT, extracted the `read:logs-*` scope, confirmed it matches the `logs-demo` index pattern, and forwarded the request to OpenSearch. Without a token, the proxy rejects the request before it reaches OpenSearch.

---

## Step 4: Inspect Your Token

See what's inside the JWT:

```bash
curl -s http://localhost:8443/oauth/introspect \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=$TOKEN" | jq
```

Expected:
```json
{
  "active": true,
  "client_id": "abc123...",
  "scope": "read:logs-*",
  "exp": 1712959200
}
```

**What happened**: The introspection endpoint decoded the token and returned its claims. This is useful for debugging — you can see exactly what scopes and identity the proxy sees.

---

## Step 5: Revoke the Token

When you're done, revoke the token:

```bash
curl -s -X POST http://localhost:8443/oauth/revoke \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=$TOKEN"
```

Expected: `200 OK` (empty body)

Verify it's revoked:
```bash
curl -s http://localhost:8443/logs-demo/_search \
  -H "Authorization: Bearer $TOKEN"
```

Expected:
```json
{"error": "token revoked"}
```

**What happened**: The proxy added the token to its revocation list. Any subsequent request with this token is rejected immediately.

---

## Step 6: Try the PKCE Flow (Browser)

For browser-based apps, use PKCE instead of client secrets:

1. Open in your browser: http://localhost:8443/demo
2. Click "Login" — this starts the PKCE flow
3. On the consent screen, review the requested scopes and click "Approve"
4. You're redirected back with an access token
5. The demo app uses the token to search logs

**What happened**: The browser generated a random `code_verifier`, hashed it to create a `code_challenge`, and sent the challenge with the authorization request. After you approved, the proxy issued an authorization code. The browser exchanged the code + original verifier for a token. This proves the same browser that started the flow is completing it — no client secret needed.

---

## Step 7: Explore the Admin API

List all registered clients:
```bash
curl -s http://localhost:8443/admin/clients | jq '.[].client_name'
```

View analytics:
```bash
curl -s http://localhost:8443/admin/analytics | jq
```

Check Cedar policies:
```bash
curl -s http://localhost:8443/admin/cedar/policies | jq
```

Trigger a key rotation:
```bash
curl -s -X POST http://localhost:8443/admin/keys/rotate | jq
```

---

## What's Next

| Goal | Resource |
|------|----------|
| Add Cedar policies for fine-grained access | [Cedar Guide](cedar-guide.md) |
| Connect to your own OpenSearch cluster | [Deployment Guide](deployment.md) |
| Set up monitoring | [Monitoring Guide](monitoring.md) |
| Use the CLI | [CLI Guide](cli-guide.md) |
| Integrate from your app | [SDK Guide](sdk-guide.md) |
| Understand the architecture | [Architecture](architecture.md) |

---

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `connection refused` on port 8443 | Is the container running? `docker ps` |
| `unknown issuer` | The token was issued by a different provider than configured |
| `scope not permitted` | Your client doesn't have the requested scope |
| `token expired` | Get a new token (Step 2) |
| PKCE consent screen doesn't appear | Clear browser cookies, try incognito |

For more: [Troubleshooting Guide](troubleshooting.md)
