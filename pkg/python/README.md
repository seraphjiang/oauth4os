# oauth4os Python SDK

Python client for [oauth4os](https://github.com/seraphjiang/oauth4os) — OAuth 2.0 proxy for OpenSearch.

## Install

```bash
pip install -e pkg/python/
# or
pip install requests  # only dependency
```

## Usage

```python
from oauth4os import Client

# Token is auto-managed — fetched on first call, refreshed when expired
c = Client("http://localhost:8443", "my-client", "my-secret", scopes=["read:logs-*"])

# Search
docs = c.search("logs-*", {"query": {"match": {"level": "error"}}})

# Index
c.index("logs-app", {"level": "info", "msg": "deployed"})

# Health
print(c.health())  # {"status": "ok", "version": "0.2.0"}

# Token lifecycle
token = c.create_token("read:logs-*")
c.revoke_token(token)

# Dynamic client registration
new_id, new_secret = c.register("my-new-agent", "read:*")

# Raw request
resp = c.do("GET", "/logs-*/_count")
```
