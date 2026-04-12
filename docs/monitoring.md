# Monitoring Guide

Set up Prometheus + Grafana monitoring for oauth4os using the included Docker Compose stack and pre-built dashboard.

---

## Quick Start

```bash
# Start proxy + monitoring stack
docker compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d
```

This starts:
- **oauth4os proxy** on `:8443`
- **Prometheus** on `:9090` — scrapes `/metrics` every 15s
- **Grafana** on `:3000` — pre-configured dashboard (login: admin/admin)

Open http://localhost:3000 → Dashboards → oauth4os.

---

## Metrics Reference

The proxy exposes Prometheus metrics at `GET /metrics`:

| Metric | Type | Description | Alert Threshold |
|--------|------|-------------|-----------------|
| `oauth4os_requests_total` | Counter | Total proxied requests | — |
| `oauth4os_requests_active` | Gauge | In-flight requests | > 50 |
| `oauth4os_requests_failed` | Counter | 4xx/5xx responses | > 5% of total |
| `oauth4os_auth_success` | Counter | Successful JWT validations | — |
| `oauth4os_auth_failed` | Counter | Failed JWT validations | Spike detection |
| `oauth4os_cedar_denied` | Counter | Cedar policy denials | — |
| `oauth4os_rate_limited` | Counter | Rate limit rejections (429) | — |
| `oauth4os_upstream_errors` | Counter | Upstream connection failures | Any increase |
| `oauth4os_uptime_seconds` | Gauge | Proxy uptime | Reset = restart |

---

## Grafana Dashboard

The pre-built dashboard (`deploy/grafana/dashboards/oauth4os.json`) includes:

| Panel | Type | Query |
|-------|------|-------|
| Requests/sec | Time series | `rate(oauth4os_requests_total[5m])` |
| Active Requests | Gauge | `oauth4os_requests_active` |
| Uptime | Stat | `oauth4os_uptime_seconds` |
| Auth Success vs Failure | Time series | `rate(oauth4os_auth_success[5m])`, `rate(oauth4os_auth_failed[5m])` |
| Error Rate | Time series | `rate(oauth4os_requests_failed[5m])` |
| Cedar Denials | Time series | `rate(oauth4os_cedar_denied[5m])` |
| Rate Limited | Time series | `rate(oauth4os_rate_limited[5m])` |
| Upstream Errors | Time series | `rate(oauth4os_upstream_errors[5m])` |

---

## Alert Rules

Pre-built alerts in `deploy/prometheus/alerts.yml`:

```yaml
groups:
  - name: oauth4os
    rules:
      - alert: HighErrorRate
        expr: rate(oauth4os_requests_failed[5m]) / rate(oauth4os_requests_total[5m]) > 0.05
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Error rate above 5%"

      - alert: HighAuthFailures
        expr: rate(oauth4os_auth_failed[5m]) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Sustained auth failures (>1/s for 5m)"

      - alert: UpstreamDown
        expr: increase(oauth4os_upstream_errors[5m]) > 5
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Upstream OpenSearch errors increasing"

      - alert: ProxyRestarted
        expr: oauth4os_uptime_seconds < 300
        for: 0m
        labels:
          severity: info
        annotations:
          summary: "Proxy restarted within last 5 minutes"
```

---

## Custom Prometheus Setup

If you're running your own Prometheus, add this scrape config:

```yaml
scrape_configs:
  - job_name: oauth4os
    scrape_interval: 15s
    static_configs:
      - targets: ['oauth4os:8443']
    metrics_path: /metrics
```

For HTTPS with self-signed certs:

```yaml
scrape_configs:
  - job_name: oauth4os
    scheme: https
    tls_config:
      insecure_skip_verify: true
    static_configs:
      - targets: ['oauth4os:8443']
```

---

## Health Endpoints

In addition to Prometheus metrics, use these for monitoring:

| Endpoint | Use Case | Response |
|----------|----------|----------|
| `GET /health` | Load balancer probe | `{"status":"ok","version":"1.0.0"}` |
| `GET /health/deep` | Monitoring dashboard | Checks upstream, JWKS, TLS cert |
| `GET /admin/analytics` | Usage dashboard | Top clients, scope distribution |

### Deep Health Example

```bash
curl -sf https://proxy:8443/health/deep | python3 -m json.tool
```

```json
{
  "status": "ok",
  "upstream": {"status": "ok", "latency_ms": 45},
  "jwks": {"status": "ok", "keys": 1, "last_refresh": "2025-04-12T06:50:00Z"}
}
```

---

## Log-Based Monitoring

The proxy writes structured JSON logs to stdout. Integrate with your log aggregator:

**CloudWatch (ECS/App Runner)**: Logs are captured automatically via the `awslogs` driver.

**Fluentd/Vector**: Parse JSON from container stdout.

**Query audit logs via API**:
```bash
curl "https://proxy:8443/admin/audit?client_id=my-agent&limit=10"
```
