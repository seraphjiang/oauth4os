# oauth4os Java SDK

Zero external dependencies — uses `java.net.http.HttpClient` (JDK 11+).

```java
var client = new OAuth4OSClient("http://localhost:8443", "my-client", "my-secret", "read:logs-*");

// Token auto-managed
String results = client.search("logs-*", "{\"query\":{\"match_all\":{}}}");
String health = client.health();
client.revokeToken("tok_abc123");
```

## Methods

- `OAuth4OSClient(url, clientId, secret, scopes)` — constructor
- `token()` — get/refresh access token
- `search(index, queryJson)` — query OpenSearch
- `index(index, docJson)` — write document
- `health()` — proxy health check
- `revokeToken(id)` — revoke token
- `register(name, scope)` — dynamic client registration
- `doRequest(method, path, body)` — raw authenticated request
