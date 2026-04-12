# oauth4os Rust SDK

```rust
use oauth4os::Client;

let client = Client::new("http://localhost:8443", "my-client", "my-secret")
    .scopes(vec!["read:logs-*"]);

// Token auto-managed
let docs = client.search("logs-*", serde_json::json!({"query": {"match_all": {}}})).unwrap();
```

## Methods

- `new(url, client_id, secret)` — constructor
- `scopes(vec)` — set scopes (builder pattern)
- `token()` — get/refresh access token
- `search(index, query)` — query OpenSearch
- `index(index, doc)` — write document
- `health()` — proxy health check
- `revoke_token(id)` — revoke token
- `register(name, scope)` — dynamic client registration

## Dependencies

`reqwest`, `serde`, `serde_json`
