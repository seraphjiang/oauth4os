# Contributing to oauth4os

Thank you for your interest in contributing! oauth4os is an open-source OAuth 2.0 proxy for OpenSearch.

## Getting Started

```bash
git clone https://github.com/seraphjiang/oauth4os.git
cd oauth4os
docker compose up -d          # Start OpenSearch + proxy
go test ./... -v              # Run all tests
```

## Development

- **Language**: Go 1.22+
- **Build**: `go build ./cmd/proxy`
- **Test**: `go test ./... -v`
- **Lint**: `golangci-lint run`
- **Integration tests**: `./test/run-integration.sh` (requires Docker)

## Pull Request Process

1. Fork the repo and create a feature branch
2. Write tests for new functionality
3. Ensure all tests pass: `go test ./... -v`
4. Update documentation if needed
5. Submit a PR with a clear description

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(proxy): Add connection pooling
fix(jwt): Handle expired JWKS cache
test(cedar): Add multi-provider policy tests
docs: Update quickstart guide
```

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep functions focused and small
- Add godoc comments for exported types and functions
- Error messages should be lowercase, no trailing punctuation

## Reporting Issues

- Use GitHub Issues for bugs and feature requests
- Include steps to reproduce for bugs
- Check existing issues before creating new ones

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
