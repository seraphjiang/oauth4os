# SDK Guide

How to integrate with oauth4os from Go, Python, Node.js, Rust, and Java. All examples use the standard OAuth 2.0 client credentials flow — no custom SDK required.

---

## Go

Using the standard `net/http` package:

```go
package main

import (
    "fmt"
    "io"
    "net/http"
    "net/url"
    "encoding/json"
    "strings"
)

const proxyURL = "https://f5cmk2hxwx.us-west-2.awsapprunner.com"

type TokenResponse struct {
    AccessToken  string `json:"access_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int    `json:"expires_in"`
    RefreshToken string `json:"refresh_token"`
}

// GetToken issues a scoped access token.
func GetToken(clientID, clientSecret, scope string) (*TokenResponse, error) {
    resp, err := http.PostForm(proxyURL+"/oauth/token", url.Values{
        "grant_type":    {"client_credentials"},
        "client_id":     {clientID},
        "client_secret": {clientSecret},
        "scope":         {scope},
    })
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    var tok TokenResponse
    json.NewDecoder(resp.Body).Decode(&tok)
    return &tok, nil
}

// Search queries OpenSearch through the proxy.
func Search(token, index, query string) ([]byte, error) {
    body := fmt.Sprintf(`{"query":{"query_string":{"query":%q}},"size":10}`, query)
    req, _ := http.NewRequest("POST", proxyURL+"/"+index+"/_search", strings.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}

func main() {
    tok, _ := GetToken("demo-agent", "demo-secret", "read:logs-*")
    results, _ := Search(tok.AccessToken, "logs-demo", "level:ERROR")
    fmt.Println(string(results))
}
```

---

## Python

Using `requests`:

```python
import requests

PROXY = "https://f5cmk2hxwx.us-west-2.awsapprunner.com"

def get_token(client_id, client_secret, scope):
    """Issue a scoped access token."""
    resp = requests.post(f"{PROXY}/oauth/token", data={
        "grant_type": "client_credentials",
        "client_id": client_id,
        "client_secret": client_secret,
        "scope": scope,
    })
    resp.raise_for_status()
    return resp.json()

def search(token, index, query, size=10):
    """Search OpenSearch through the proxy."""
    resp = requests.post(
        f"{PROXY}/{index}/_search",
        headers={"Authorization": f"Bearer {token}"},
        json={"query": {"query_string": {"query": query}}, "size": size},
    )
    resp.raise_for_status()
    return resp.json()

def introspect(token):
    """Check if a token is active (RFC 7662)."""
    resp = requests.post(f"{PROXY}/oauth/introspect", data={"token": token})
    return resp.json()

# Usage
tok = get_token("demo-agent", "demo-secret", "read:logs-*")
results = search(tok["access_token"], "logs-demo", "level:ERROR")
for hit in results["hits"]["hits"]:
    src = hit["_source"]
    print(f"[{src['level']}] {src['service']}: {src['message']}")
```

### With auto-refresh

```python
import time

class OAuth4OSClient:
    def __init__(self, proxy, client_id, client_secret, scope):
        self.proxy = proxy
        self.client_id = client_id
        self.client_secret = client_secret
        self.scope = scope
        self.token = None
        self.expires_at = 0

    def _ensure_token(self):
        if time.time() < self.expires_at - 60:  # refresh 60s before expiry
            return
        tok = get_token(self.client_id, self.client_secret, self.scope)
        self.token = tok["access_token"]
        self.expires_at = time.time() + tok["expires_in"]

    def search(self, index, query, size=10):
        self._ensure_token()
        return search(self.token, index, query, size)

client = OAuth4OSClient(PROXY, "demo-agent", "demo-secret", "read:logs-*")
results = client.search("logs-demo", "service:payment AND level:ERROR")
```

---

## Node.js

Using built-in `fetch` (Node 18+):

```javascript
const PROXY = 'https://f5cmk2hxwx.us-west-2.awsapprunner.com';

async function getToken(clientId, clientSecret, scope) {
  const resp = await fetch(`${PROXY}/oauth/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      grant_type: 'client_credentials',
      client_id: clientId,
      client_secret: clientSecret,
      scope,
    }),
  });
  if (!resp.ok) throw new Error(`Token error: ${resp.status}`);
  return resp.json();
}

async function search(token, index, query, size = 10) {
  const resp = await fetch(`${PROXY}/${index}/_search`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      query: { query_string: { query } },
      size,
    }),
  });
  if (!resp.ok) throw new Error(`Search error: ${resp.status}`);
  return resp.json();
}

// Usage
const tok = await getToken('demo-agent', 'demo-secret', 'read:logs-*');
const results = await search(tok.access_token, 'logs-demo', 'level:ERROR');
results.hits.hits.forEach(h => {
  const s = h._source;
  console.log(`[${s.level}] ${s.service}: ${s.message}`);
});
```

### Browser (PKCE flow)

For browser apps, use the PKCE authorization code flow instead of client credentials:

```javascript
async function startPKCELogin(clientId, redirectUri, scope) {
  // Generate verifier + challenge
  const verifier = Array.from(crypto.getRandomValues(new Uint8Array(32)),
    b => b.toString(16).padStart(2, '0')).join('');
  sessionStorage.setItem('pkce_verifier', verifier);

  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(verifier));
  const challenge = btoa(String.fromCharCode(...new Uint8Array(digest)))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');

  // Redirect to authorize
  location.href = `${PROXY}/oauth/authorize?` + new URLSearchParams({
    response_type: 'code',
    client_id: clientId,
    redirect_uri: redirectUri,
    code_challenge: challenge,
    code_challenge_method: 'S256',
    scope,
  });
}

// On callback page, exchange code for token:
async function handleCallback() {
  const code = new URLSearchParams(location.search).get('code');
  const verifier = sessionStorage.getItem('pkce_verifier');
  const resp = await fetch(`${PROXY}/oauth/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      grant_type: 'authorization_code',
      code,
      code_verifier: verifier,
      redirect_uri: location.origin + '/callback',
    }),
  });
  return resp.json();
}
```

---

## Rust

Using `reqwest`:

```rust
use reqwest::Client;
use serde::Deserialize;
use std::collections::HashMap;

const PROXY: &str = "https://f5cmk2hxwx.us-west-2.awsapprunner.com";

#[derive(Deserialize)]
struct TokenResponse {
    access_token: String,
    expires_in: u64,
}

async fn get_token(client_id: &str, client_secret: &str, scope: &str)
    -> Result<TokenResponse, reqwest::Error>
{
    let mut params = HashMap::new();
    params.insert("grant_type", "client_credentials");
    params.insert("client_id", client_id);
    params.insert("client_secret", client_secret);
    params.insert("scope", scope);

    Client::new()
        .post(format!("{PROXY}/oauth/token"))
        .form(&params)
        .send()
        .await?
        .json()
        .await
}

async fn search(token: &str, index: &str, query: &str)
    -> Result<serde_json::Value, reqwest::Error>
{
    Client::new()
        .post(format!("{PROXY}/{index}/_search"))
        .bearer_auth(token)
        .json(&serde_json::json!({
            "query": {"query_string": {"query": query}},
            "size": 10
        }))
        .send()
        .await?
        .json()
        .await
}

#[tokio::main]
async fn main() {
    let tok = get_token("demo-agent", "demo-secret", "read:logs-*")
        .await.unwrap();
    let results = search(&tok.access_token, "logs-demo", "level:ERROR")
        .await.unwrap();
    println!("{}", serde_json::to_string_pretty(&results).unwrap());
}
```

---

## Java

Using `java.net.http` (Java 11+):

```java
import java.net.URI;
import java.net.http.*;
import java.net.http.HttpRequest.BodyPublishers;
import java.net.http.HttpResponse.BodyHandlers;

public class OAuth4OSClient {
    static final String PROXY = "https://f5cmk2hxwx.us-west-2.awsapprunner.com";
    static final HttpClient client = HttpClient.newHttpClient();

    public static String getToken(String clientId, String clientSecret, String scope)
            throws Exception {
        var body = String.format(
            "grant_type=client_credentials&client_id=%s&client_secret=%s&scope=%s",
            clientId, clientSecret, scope);
        var req = HttpRequest.newBuilder()
            .uri(URI.create(PROXY + "/oauth/token"))
            .header("Content-Type", "application/x-www-form-urlencoded")
            .POST(BodyPublishers.ofString(body))
            .build();
        var resp = client.send(req, BodyHandlers.ofString());
        // Parse access_token from JSON response
        var json = resp.body();
        int start = json.indexOf("\"access_token\":\"") + 16;
        int end = json.indexOf("\"", start);
        return json.substring(start, end);
    }

    public static String search(String token, String index, String query)
            throws Exception {
        var body = String.format(
            "{\"query\":{\"query_string\":{\"query\":\"%s\"}},\"size\":10}", query);
        var req = HttpRequest.newBuilder()
            .uri(URI.create(PROXY + "/" + index + "/_search"))
            .header("Authorization", "Bearer " + token)
            .header("Content-Type", "application/json")
            .POST(BodyPublishers.ofString(body))
            .build();
        return client.send(req, BodyHandlers.ofString()).body();
    }

    public static void main(String[] args) throws Exception {
        var token = getToken("demo-agent", "demo-secret", "read:logs-*");
        var results = search(token, "logs-demo", "level:ERROR");
        System.out.println(results);
    }
}
```

---

## curl (Shell)

For scripts and one-liners:

```bash
# Get token
TOKEN=$(curl -sf -X POST "$PROXY/oauth/token" \
  -d "grant_type=client_credentials&client_id=demo-agent&client_secret=demo-secret&scope=read:logs-*" \
  | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

# Search
curl -sf -H "Authorization: Bearer $TOKEN" \
  "$PROXY/logs-demo/_search" \
  -H "Content-Type: application/json" \
  -d '{"query":{"term":{"level":"ERROR"}},"size":5}'

# Introspect
curl -sf -X POST "$PROXY/oauth/introspect" -d "token=$TOKEN"

# Revoke
curl -sf -X DELETE "$PROXY/oauth/token/$TOKEN_ID"
```

---

## Common Patterns

### Token caching

All SDKs should cache tokens and refresh before expiry:

```
if token is null OR current_time > (expires_at - 60 seconds):
    token = get_new_token()
    expires_at = current_time + token.expires_in
```

### Error handling

Check for these HTTP status codes:

| Status | Meaning | Action |
|--------|---------|--------|
| 401 | Token expired/invalid | Get a new token and retry |
| 403 | Scope insufficient or Cedar denied | Request broader scope or check policies |
| 429 | Rate limited | Wait for `Retry-After` seconds, then retry |
| 502 | Upstream error | Retry with backoff |

### Refresh tokens

For long-running processes, use refresh tokens instead of re-authenticating:

```
if access_token expired:
    new_tokens = POST /oauth/token (grant_type=refresh_token, refresh_token=RT)
    access_token = new_tokens.access_token
    refresh_token = new_tokens.refresh_token  # rotated!
```

Note: refresh tokens are single-use. Each refresh returns a new refresh token. Reusing an old refresh token revokes the entire token family (security feature).
