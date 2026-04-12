# Contributing to oauth4os

Thanks for your interest! This guide helps you get started.

## Development Setup

```bash
git clone https://github.com/seraphjiang/oauth4os.git
cd oauth4os
make build        # Build proxy + CLI
make lint         # Run linters
make test-unit    # Run unit tests
```

## Running Locally

```bash
docker compose up                          # Proxy + OpenSearch + Dashboards
docker compose -f docker-compose.monitoring.yml up  # + Prometheus + Grafana
```

## Testing

```bash
make test-unit          # Unit tests
make test-integration   # Integration tests (needs Docker)
make test-e2e           # E2E tests (needs docker compose up)
```

## Pull Requests

1. Fork the repo and create a feature branch
2. Write tests for new functionality
3. Run `make lint` and `make test-unit` before submitting
4. Follow [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` new feature
   - `fix:` bug fix
   - `docs:` documentation
   - `test:` tests
   - `ci:` CI/CD changes
   - `chore:` maintenance

## Project Structure

```
cmd/proxy/       — Proxy server entry point
cmd/cli/         — CLI tool
internal/        — Core packages (jwt, cedar, scope, token, audit, tracing, logging)
deploy/          — Helm, CDK, Prometheus, Grafana configs
test/            — Integration, E2E, chaos tests
docs/            — Architecture, security, API spec, guides
```

## Code Style

- `gofmt` and `golangci-lint` enforced in CI
- Keep packages small and focused
- Error messages should not leak internal details
- Security-sensitive code (internal/jwt/, internal/cedar/, internal/token/) requires extra review

## Reporting Issues

Use [GitHub Issues](https://github.com/seraphjiang/oauth4os/issues). Include:
- Steps to reproduce
- Expected vs actual behavior
- Go version, OS, OpenSearch version

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
