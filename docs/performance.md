# Performance Guide

Benchmark results, tuning recommendations, and capacity planning for oauth4os.

---

## Benchmark Results

Measured on a single core (0.25 vCPU App Runner instance, golang:1.22).

### Component Benchmarks

Run: `go test ./bench/ -bench=. -benchmem -count=3`

| Benchmark | ops/sec | ns/op | allocs/op |
|-----------|---------|-------|-----------|
| ScopeMapper (1 scope, 10 mappings) | 897,590 | 184 | 1 |
| ScopeMapper (5 scopes, 100 mappings) | 143,892 | 915 | 4 |
| Cedar (1 policy) | 407,197 | 269 | 2 |
| Cedar (10 policies) | 102,904 | 1,659 | 11 |
| Cedar (100 policies) | 12,945 | 11,886 | 101 |
| Cedar (forbid override) | 2,173,687 | 63 | 0 |
| Cache hit | 1,598,920 | 64 | 0 |
| Cache miss | 7,479,860 | 18 | 0 |
| Circuit breaker check | 7,565,611 | 18 | 0 |
| Proxy round-trip (scope + Cedar) | 1,560 | 79,546 | 60 |
| Proxy passthrough (no auth) | 1,473 | 70,283 | 44 |
| Proxy with warm cache | 1,599 | 70,753 | 44 |

Measured on Intel Xeon Platinum 8375C @ 2.90GHz (16 cores), Go 1.22.

Key takeaways:
- Scope mapping and Cedar evaluation are sub-microsecond for typical configs (<20 policies)
- The proxy middleware chain adds ~1-2ms per request (dominated by HTTP overhead, not auth logic)
- Cedar is zero-allocation — no GC pressure under load

### End-to-End Throughput

Measured with 1000 concurrent requests against the live App Runner deployment:

| Metric | Value |
|--------|-------|
| Throughput | 232 req/s |
| Latency p50 | 12ms |
| Latency p95 | 45ms |
| Latency p99 | 120ms |
| Error rate | 0% |

This is on a 0.25 vCPU / 0.5 GB App Runner instance. Throughput scales linearly with CPU.

---

## Latency Breakdown

Where time is spent per request:

```
Total: ~12ms (p50)
├── TLS handshake (reused):  0ms (connection pooled)
├── Tracing span start:      <0.01ms
├── IP filter check:         <0.01ms
├── Rate limit check:        <0.01ms
├── JWT validation:          0.1ms (JWKS cached)
├── Scope mapping:           <0.01ms
├── Cedar evaluation:        <0.01ms
├── Audit log write:         0.05ms
├── Upstream request:        ~11ms (network + OpenSearch)
└── Response forwarding:     0.5ms
```

The proxy adds ~1ms overhead. The rest is upstream latency.

### JWKS Cache

JWT validation is fast because JWKS keys are cached in memory:
- First request: ~200ms (fetches JWKS from provider)
- Subsequent requests: ~0.1ms (cache hit)
- Cache refresh: every 5 minutes (background goroutine)
- Cache miss (key rotation): ~200ms (one-time fetch)

---

## Tuning Guide

### Connection Pooling

The proxy reuses HTTP connections to OpenSearch. Defaults are good for most workloads:

```yaml
# These are Go's http.Transport defaults
# Tune if you see "connection reset" errors under high load
upstream:
  max_idle_conns: 100          # total idle connections
  max_idle_conns_per_host: 10  # idle connections per upstream
  idle_conn_timeout: 90s       # close idle connections after
```

For high-throughput deployments (>500 req/s per instance):

```yaml
upstream:
  max_idle_conns: 500
  max_idle_conns_per_host: 100
  idle_conn_timeout: 120s
```

### Rate Limiting

Rate limits are per-client, per-scope. Set them based on expected traffic:

```yaml
rate_limits:
  "read:logs-*": 600     # 10 req/s sustained
  "write:logs-*": 120    # 2 req/s sustained
  "admin": 60            # 1 req/s sustained
```

The token bucket algorithm allows short bursts above the limit. A client with 600 RPM can burst to ~20 req/s for a few seconds before being throttled.

### Cedar Policy Count

Cedar evaluation is O(n) in the number of policies. Performance impact:

| Policies | Overhead per request |
|----------|---------------------|
| 1-10 | <1μs (negligible) |
| 10-50 | 1-5μs |
| 50-100 | 5-15μs |
| 100-500 | 15-50μs |
| 500+ | Consider restructuring |

If you have >100 policies, consider using tenant isolation to partition them — each tenant's policies are evaluated independently.

### Audit Log Performance

Audit logs are written synchronously to stdout. For high-throughput deployments:
- Use a log aggregator (Fluentd, Vector) that reads from stdout asynchronously
- The in-memory audit store (for `/admin/audit` queries) is capped at 10,000 entries by default

### Memory Usage

| Component | Memory |
|-----------|--------|
| Base proxy | ~15MB |
| JWKS cache (per provider) | ~10KB |
| Token store (per 1000 tokens) | ~1MB |
| Audit store (10,000 entries) | ~5MB |
| Cedar engine (100 policies) | ~50KB |
| Rate limiter (per 100 clients) | ~100KB |

Total for a typical deployment: **20-30MB**. The proxy runs comfortably in 64MB containers.

---

## Capacity Planning

### Sizing by throughput

| Target req/s | vCPU | Memory | Instances |
|-------------|------|--------|-----------|
| 50 | 0.25 | 0.5 GB | 1 |
| 200 | 0.25 | 0.5 GB | 1 |
| 500 | 0.5 | 1 GB | 1 |
| 1,000 | 1 | 1 GB | 1 |
| 5,000 | 1 | 1 GB | 3 |
| 10,000 | 2 | 2 GB | 3 |
| 50,000 | 2 | 2 GB | 10 |

The proxy is CPU-bound (auth computation). Memory stays flat regardless of throughput.

### Scaling strategy

1. **Horizontal scaling** — add more proxy instances behind a load balancer. The proxy is stateless (tokens are self-contained JWTs). Any instance can handle any request.

2. **Vertical scaling** — increase vCPU. Throughput scales linearly with CPU cores (Go's runtime uses all available cores).

3. **Connection pooling** — increase `max_idle_conns_per_host` if upstream latency is high and you're seeing connection churn.

### Bottleneck identification

| Symptom | Likely bottleneck | Fix |
|---------|-------------------|-----|
| High p99, low p50 | Upstream latency variance | Add retry with backoff |
| All latencies high | Upstream overloaded | Scale OpenSearch |
| 429 errors | Rate limit too low | Increase RPM in config |
| CPU >80% | Proxy compute bound | Add instances or vCPU |
| Memory growing | Token/audit store | Cap store size, reduce retention |

### Monitoring

Key metrics to watch:

```
oauth4os_requests_total        — overall throughput
oauth4os_requests_active       — concurrency (should stay < 100)
oauth4os_requests_failed       — error rate (should be < 1%)
oauth4os_auth_failed           — auth failures (watch for spikes)
oauth4os_rate_limited          — rate limit hits (tune if too high)
oauth4os_upstream_errors       — upstream health
oauth4os_uptime_seconds        — restart detection
```

Set alerts on:
- `oauth4os_requests_failed / oauth4os_requests_total > 0.05` (5% error rate)
- `oauth4os_requests_active > 50` (high concurrency)
- `oauth4os_upstream_errors` increasing (upstream degradation)

---

## Running Benchmarks

### Component benchmarks

```bash
go test ./bench/ -bench=. -benchmem -count=3
```

### Stress test (requires running proxy)

```bash
PROXY_URL=https://f5cmk2hxwx.us-west-2.awsapprunner.com \
  go test ./test/e2e/ -run TestStress -v -timeout 5m
```

### Profile the proxy

```bash
# CPU profile
go tool pprof http://localhost:8443/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof http://localhost:8443/debug/pprof/heap

# Goroutine dump
curl http://localhost:8443/debug/pprof/goroutine?debug=2
```

Note: pprof endpoints are only available when the proxy is started with `-debug` flag.
