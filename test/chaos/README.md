# Chaos Tests

Fault injection tests for oauth4os proxy resilience.

## Prerequisites

```bash
docker compose -f docker-compose.test.yml up -d --wait
export PROXY_URL=http://localhost:8443
```

## Tests

| Script | What it tests |
|--------|--------------|
| `kill-upstream.sh` | Kill OpenSearch mid-request — proxy returns 502/503, doesn't crash |
| `expired-jwks.sh` | Serve empty/broken JWKS — proxy rejects tokens gracefully |
| `clock-skew.sh` | Expired tokens + missing exp claim — proxy enforces expiry |

## Run all

```bash
for f in test/chaos/*.sh; do bash "$f"; echo; done
```
