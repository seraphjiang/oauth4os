# Contributing to oauth4os

## Try It First

```bash
# Run the latest release (no build needed)
docker run -p 8443:8443 jianghuan/oauth4os:latest

# Or visit the live demo
open https://f5cmk2hxwx.us-west-2.awsapprunner.com
```

## Dev Setup

```bash
# Clone
git clone https://github.com/seraphjiang/oauth4os.git
cd oauth4os

# Build (requires Go 1.22+)
go build ./cmd/proxy/ ./cmd/cli/

# Or with Docker (no local Go needed)
docker build -t oauth4os .

# Test
go test ./internal/... -count=1

# Vet
go vet ./...
```

## Run Locally

```bash
# With Docker Compose (proxy + OpenSearch + Dashboards)
docker compose up

# Or standalone (needs config.yaml)
go run ./cmd/proxy/ -config config.yaml
```

## Project Structure

```
cmd/proxy/          Main proxy binary
cmd/cli/            CLI tool
internal/           All packages (audit, cedar, config, jwt, sigv4, etc.)
test/               Integration + E2E tests
examples/           Fluent Bit, LangChain, K8s sidecar
plugins/            OpenSearch Dashboards plugin
.github/workflows/  CI + release pipelines
```

## PR Workflow

1. Fork the repo
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make changes, ensure `go build` and `go test ./internal/...` pass
4. Commit using [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` new feature
   - `fix:` bug fix
   - `docs:` documentation
   - `ci:` CI changes
   - `test:` test changes
5. Open a PR against `main`

## Testing

```bash
# Unit tests (fast, no external deps)
go test ./internal/... -count=1

# With Docker (matches CI)
docker run --rm -v $(pwd):/app -w /app golang:1.22 \
  sh -c 'go test -buildvcs=false ./internal/...'

# E2E (requires running proxy + OpenSearch)
docker compose up -d
go test ./test/e2e/ -count=1
```

## Code Style

- `go vet` must pass
- Keep imports sorted (stdlib, then external)
- No external dependencies beyond `golang-jwt/jwt` and `gopkg.in/yaml.v3`
- Minimal code — avoid verbose implementations

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
