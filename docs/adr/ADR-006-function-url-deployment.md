# ADR-006: Lambda Function URL as Deployment Option

**Status**: Accepted
**Date**: 2026-04-12

## Context

Need a lightweight deployment option alongside ECS Fargate. Options: API Gateway + Lambda, Lambda Function URL, CloudFront Functions.

## Decision

Support Lambda Function URL as a zero-infrastructure deployment option.

## Rationale

- Function URL provides HTTPS endpoint with no API Gateway cost or config
- Go compiles to a single binary — ideal for Lambda's provided.al2023 runtime
- ARM64 (Graviton) gives best price/performance — ~200ms cold start
- ~$0.50/month for 100K requests vs ~$15/month for ECS Fargate
- API Gateway rejected for this use case: adds cost and latency for a simple proxy
- CloudFront Functions rejected: too limited (no network calls, 10KB code limit)

## Consequences

- 15-minute timeout limit — not suitable for long-running queries
- No WebSocket support — Dashboards proxy requires ECS/container deployment
- No built-in WAF — use CloudFront in front of Function URL if WAF needed
