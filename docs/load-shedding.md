# Load Shedding

oauth4os rejects requests with `503 Service Unavailable` when the proxy is overloaded.

## How It Works

```
Request → Check active count → Under threshold? → Process
                                  ↓ Over threshold
                              503 + Retry-After: 5
```

The shedder tracks concurrent in-flight requests using an atomic counter. When active requests exceed the configured threshold, new requests are immediately rejected with:

```json
{
  "error": "overloaded",
  "active": 150,
  "threshold": 100
}
```

## Configuration

```yaml
# config.yaml
load_shedding:
  max_concurrent: 100  # default: 100
```

## Response Headers

| Header | Value | Description |
|--------|-------|-------------|
| `Retry-After` | `5` | Seconds to wait before retrying |
| `Content-Type` | `application/json` | Error body format |

## Monitoring

| Metric | Description |
|--------|-------------|
| `oauth4os_loadshed_rejected` | Total rejected requests |
| `oauth4os_active_requests` | Current in-flight requests |

## Client Handling

Clients should implement exponential backoff when receiving 503:

```python
import time, requests

def query_with_backoff(url, headers, max_retries=3):
    for attempt in range(max_retries):
        resp = requests.get(url, headers=headers)
        if resp.status_code != 503:
            return resp
        wait = int(resp.headers.get('Retry-After', 5))
        time.sleep(wait * (2 ** attempt))
    raise Exception("Service overloaded")
```

## Tuning

- **Too low**: Legitimate requests rejected during normal load
- **Too high**: Proxy OOMs or upstream overwhelmed
- **Recommended**: Set to 2-3x your expected peak concurrent requests
- Monitor `oauth4os_active_requests` to find your baseline
