# Monitoring with Prometheus + Grafana

## 1. Scrape /metrics

oauth4os exposes all metrics at `GET /metrics` in Prometheus exposition format.

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: oauth4os
    scrape_interval: 15s
    static_targets:
      - targets: ['localhost:8443']
    scheme: https
    tls_config:
      insecure_skip_verify: true  # remove in production
    metrics_path: /metrics
```

Reload Prometheus: `curl -X POST http://localhost:9090/-/reload`

### Available metrics

| Metric | Type | Description |
|---|---|---|
| `oauth4os_requests_total` | counter | Total proxy requests |
| `oauth4os_requests_active` | gauge | Currently active requests |
| `oauth4os_auth_success` | counter | Successful authentications |
| `oauth4os_auth_failed` | counter | Failed authentications |
| `oauth4os_rate_limited` | counter | Rate-limited requests |
| `oauth4os_cache_hits` / `cache_misses` | counter | Response cache performance |
| `oauth4os_circuit_opens` | counter | Circuit breaker activations |
| `oauth4os_upstream_healthy` | gauge | Upstream health (1/0) |
| `oauth4os_upstream_latency_ms` | gauge | Health check latency |
| `oauth4os_request_duration_seconds` | histogram | Latency by endpoint (p50/p95/p99) |
| `oauth4os_http_requests_total` | counter | Requests by method + path |
| `oauth4os_http_request_duration` | summary | Latency summary by method + path |
| `oauth4os_metric_cardinality` | gauge | Unique label combinations per metric |

## 2. Push metrics via remote write

External apps can push metrics to oauth4os:

```bash
curl -X POST https://localhost:8443/api/v1/write \
  -H 'Content-Type: application/json' \
  -d '{
    "timeseries": [{
      "labels": {"__name__": "myapp_errors_total", "service": "payment"},
      "samples": [{"value": 42}]
    }]
  }'
```

Pushed metrics appear on `/metrics` alongside internal metrics. Series cap: 10,000.

## 3. Import the Grafana dashboard

1. Open Grafana → Dashboards → Import
2. Upload `deploy/grafana/dashboards/oauth4os.json`
3. Select your Prometheus data source
4. Click Import

The dashboard includes 18 panels: request rates, error rates, cache hit ratio, circuit breaker status, latency percentiles (p50/p95/p99), and per-endpoint breakdown.

## 4. Set up alerting rules

Copy the rules file and reference it in Prometheus:

```bash
cp deploy/prometheus/alerts.yml /etc/prometheus/rules/oauth4os.yml
```

Add to `prometheus.yml`:

```yaml
rule_files:
  - /etc/prometheus/rules/oauth4os.yml
```

### 10 alerting rules included

| Alert | Condition | Severity |
|---|---|---|
| HighErrorRate | >10% requests failing, 5m | critical |
| HighLatency | upstream >2s, 5m | warning |
| CircuitBreakerOpen | circuit opens >0, 1m | critical |
| UpstreamDown | healthy=0, 2m | critical |
| HighRateLimit | rate limited >100/min, 5m | warning |
| CacheHitRateLow | hit rate <50%, 10m | info |
| HighAuthFailure | auth failures >50/min, 5m | warning |
| HighActiveConnections | >500 active, 3m | warning |
| HighP99Latency | p99 >2s, 3m | warning |
| HighP50Latency | median >500ms, 5m | info |

## 5. Docker Compose (quickstart)

```bash
docker compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d
```

This starts oauth4os + Prometheus + Grafana. Access:
- oauth4os: https://localhost:8443
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)
