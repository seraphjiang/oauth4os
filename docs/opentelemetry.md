# OpenTelemetry Integration

Export traces, metrics, and logs from oauth4os to any OTel-compatible backend.

## Quick Start

```yaml
# config.yaml
tracing:
  otlp_endpoint: http://otel-collector:4318/v1/traces
  sample_rate: 1.0  # 1.0 = all requests, 0.1 = 10%
  service_name: oauth4os
```

## Exported Spans

Every proxied request generates spans for each phase:

| Span | Attributes |
|------|-----------|
| `oauth4os.receive` | `http.method`, `http.url`, `http.request_id` |
| `oauth4os.jwt_validate` | `jwt.algorithm`, `jwt.issuer`, `jwt.cached` |
| `oauth4os.scope_check` | `authz.scope`, `authz.mapped_role`, `authz.index_pattern` |
| `oauth4os.cedar_eval` | `cedar.policies_evaluated`, `cedar.decision`, `cedar.effect` |
| `oauth4os.rate_limit` | `ratelimit.bucket`, `ratelimit.remaining`, `ratelimit.limit` |
| `oauth4os.sigv4_sign` | `aws.service`, `aws.region` |
| `oauth4os.upstream` | `http.status_code`, `net.peer.name`, `tls.version` |
| `oauth4os.response` | `http.status_code`, `http.response_content_length` |

## Docker Compose with OTel Collector

```yaml
# Add to docker-compose.monitoring.yml
services:
  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.96.0
    volumes:
      - ./deploy/otel/config.yaml:/etc/otelcol/config.yaml:ro
    ports:
      - "4317:4317"   # gRPC
      - "4318:4318"   # HTTP
      - "8889:8889"   # Prometheus exporter
    depends_on:
      - oauth-proxy
```

## OTel Collector Config

```yaml
# deploy/otel/config.yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 5s
    send_batch_size: 100

exporters:
  # Jaeger
  otlp/jaeger:
    endpoint: jaeger:4317
    tls:
      insecure: true

  # Prometheus (metrics)
  prometheus:
    endpoint: 0.0.0.0:8889

  # Console (debug)
  logging:
    loglevel: info

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/jaeger, logging]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [prometheus]
```

## Backends

| Backend | Exporter | Config |
|---------|----------|--------|
| Jaeger | `otlp/jaeger` | `endpoint: jaeger:4317` |
| Zipkin | `zipkin` | `endpoint: http://zipkin:9411/api/v2/spans` |
| Grafana Tempo | `otlp` | `endpoint: tempo:4317` |
| AWS X-Ray | `awsxray` | Region auto-detected from IAM |
| Datadog | `datadog` | `api.key: DD_API_KEY` |
| Honeycomb | `otlp` | `endpoint: api.honeycomb.io:443` |

## Metrics Exported

| Metric | Type | Description |
|--------|------|-------------|
| `oauth4os_requests_total` | Counter | Total proxied requests |
| `oauth4os_request_duration_seconds` | Histogram | Request latency |
| `oauth4os_auth_success` | Counter | Successful authentications |
| `oauth4os_auth_failed` | Counter | Failed authentications |
| `oauth4os_cedar_denied` | Counter | Cedar policy denials |
| `oauth4os_tokens_active` | Gauge | Currently active tokens |
| `oauth4os_upstream_latency_seconds` | Histogram | Upstream response time |
| `oauth4os_cache_hits` | Counter | Response cache hits |
| `oauth4os_cache_misses` | Counter | Response cache misses |
