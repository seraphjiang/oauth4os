# oauth4os Performance Benchmarks

Benchmark results and scaling guidance for oauth4os proxy components.

## Running Benchmarks

```bash
go test ./bench/ -bench=. -benchmem -count=3
```

Or via Docker (no local Go required):

```bash
docker run --rm -v $(pwd):/app -w /app golang:1.22 \
  go test ./bench/ -bench=. -benchmem -count=3
```

## Component Benchmarks

### Scope Mapper

Maps OAuth scopes to OpenSearch backend roles. O(scopes × mapping_size) with deduplication.

| Benchmark | Expected Latency | Allocs |
|---|---|---|
| 1 scope, 10 mappings | ~100-200 ns/op | 2-3 allocs |
| 5 scopes, 100 mappings | ~400-800 ns/op | 5-8 allocs |
| Miss (no match) | ~50-100 ns/op | 1 alloc |

Scope mapping is negligible overhead — sub-microsecond even with large mapping tables.

### Cedar Policy Engine

Evaluates Cedar policies for fine-grained access control. Linear scan over policies with glob matching.

| Benchmark | Expected Latency | Allocs |
|---|---|---|
| 1 policy (permit) | ~200-500 ns/op | 2-4 allocs |
| 10 policies (last match) | ~2-5 µs/op | 10-20 allocs |
| 100 policies (last match) | ~20-50 µs/op | 100-200 allocs |
| Deny all (no match) | ~2-5 µs/op | 10-20 allocs |
| Forbid override | ~300-600 ns/op | 3-5 allocs |

Cedar evaluation scales linearly with policy count. For most deployments (<20 policies), overhead is under 5µs.

### Proxy Round-Trip

Full HTTP request through auth middleware (scope mapping + Cedar evaluation, excluding JWT crypto).

| Benchmark | Expected Latency | Notes |
|---|---|---|
| Auth middleware path | ~30-60 µs/op | Includes HTTP overhead |
| Passthrough (no auth) | ~15-30 µs/op | Baseline HTTP cost |

Auth overhead is ~15-30µs per request on top of baseline HTTP.

## Scaling Guidance

### Throughput Estimates

Based on single-core performance. oauth4os is stateless — scale horizontally.

| Target RPS | CPU Cores | Memory | Notes |
|---|---|---|---|
| 100 req/s | 1 | 64 MB | Single container, minimal |
| 1,000 req/s | 2 | 128 MB | Single container, comfortable |
| 10,000 req/s | 4-8 | 256 MB | 2-4 replicas recommended |
| 100,000 req/s | 16-32 | 1 GB | 8-16 replicas behind LB |

### Bottlenecks by Component

1. **JWT validation** (~100-500µs): RSA signature verification dominates. JWKS keys are cached — first request per key is slower (HTTP fetch).
2. **Cedar evaluation** (~1-50µs): Linear in policy count. Keep policies under 50 for sub-10µs.
3. **Scope mapping** (~0.1-1µs): Negligible.
4. **Reverse proxy** (~0.5-2ms): Network latency to OpenSearch dominates total request time.

### Resource Sizing

| Deployment | Replicas | CPU (per pod) | Memory (per pod) | Policies |
|---|---|---|---|---|
| Dev/test | 1 | 0.25 vCPU | 64 MB | <10 |
| Small team (100 rps) | 1 | 0.5 vCPU | 128 MB | <20 |
| Production (1K rps) | 2 | 1 vCPU | 256 MB | <50 |
| High-traffic (10K rps) | 4 | 2 vCPU | 512 MB | <100 |

### Optimization Tips

- **JWKS caching**: Default 5-minute TTL. Increase for stable providers to reduce latency spikes.
- **Connection pooling**: Proxy reuses connections to OpenSearch. Default pool size handles 1K concurrent.
- **Cedar policies**: Use specific `Equals` matches over `Pattern` (glob) where possible — 2-3x faster.
- **Scope mapping**: Flat map lookup — no optimization needed.

## Latency Budget (typical request)

```
Client → Proxy (network)     ~1ms
  JWT validation              ~0.2ms (cached key)
  Scope mapping               ~0.001ms
  Cedar evaluation            ~0.005ms (10 policies)
  Audit logging               ~0.01ms
Proxy → OpenSearch (network)  ~1-5ms
OpenSearch query              ~10-500ms
Total overhead from proxy:    ~0.2-0.5ms
```

The proxy adds <1ms overhead to every request. OpenSearch query time dominates.

## Resilience Layer Benchmarks (v0.5.0)

| Component | Operation | Throughput | Latency | Allocs |
|---|---|---|---|---|
| Response cache | GET hit | 11M ops/s | 109ns | 0 |
| Response cache | GET miss | 39M ops/s | 26ns | 0 |
| Response cache | Concurrent read | 7M ops/s | 208ns | 1 |
| Circuit breaker | Allow (closed) | 47M ops/s | 24ns | 0 |
| Circuit breaker | Allow (open) | 17M ops/s | 64ns | 0 |
| Circuit breaker | Record | 52M ops/s | 23ns | 0 |
| Circuit breaker | Concurrent | 12M ops/s | 106ns | 0 |
| Retry transport | No retry (200) | 3.2M ops/s | 332ns | 1 |

### Overhead comparison

| Layer | Added latency | Impact |
|---|---|---|
| Response cache | +75ns/request | <0.01% of typical request |
| Circuit breaker | +47ns/request | <0.01% of typical request |
| Both combined | ~120ns/request | Negligible |

All resilience layers are zero-allocation on hot paths.

## Histogram Metrics (v1.1.0)

| Component | Operation | Throughput | Latency | Allocs |
|---|---|---|---|---|
| Histogram | Observe (concurrent) | 8M ops/s | 191ns | 0 |
| ETag | Small body (16B) | 176K ops/s | 6.5µs | 22 |
| ETag | Large body (4.6KB) | 117K ops/s | 15.8µs | 21 |
| ETag | 304 cache hit | 270K ops/s | 5.3µs | 17 |

### Prometheus Metrics Summary

| Metric | Type | Description |
|---|---|---|
| oauth4os_request_duration_seconds | histogram | Request latency with per-endpoint breakdown |
| oauth4os_requests_total | counter | Total requests |
| oauth4os_requests_active | gauge | Currently active requests |
| oauth4os_auth_success / auth_failed | counter | Authentication outcomes |
| oauth4os_cedar_denied | counter | Cedar policy denials |
| oauth4os_rate_limited | counter | Rate-limited requests |
| oauth4os_cache_hits / cache_misses | counter | Response cache performance |
| oauth4os_circuit_opens | counter | Circuit breaker activations |
| oauth4os_upstream_latency_ms | gauge | Background health check latency |
| oauth4os_upstream_healthy | gauge | Upstream health (1/0) |
| oauth4os_loadshed_inflight | gauge | Active connections |
| oauth4os_loadshed_total | counter | Load-shed rejections |
