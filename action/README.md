# oauth4os GitHub Action

Get a scoped OAuth token from your oauth4os proxy in CI/CD pipelines. Use it to query OpenSearch securely from GitHub Actions.

## Usage

```yaml
- name: Get OpenSearch token
  id: auth
  uses: seraphjiang/oauth4os/action@main
  with:
    proxy-url: ${{ secrets.OAUTH4OS_URL }}
    client-id: ${{ secrets.OAUTH4OS_CLIENT_ID }}
    client-secret: ${{ secrets.OAUTH4OS_CLIENT_SECRET }}
    scope: 'read:logs-*'

- name: Query OpenSearch
  run: |
    curl -H "Authorization: Bearer ${{ steps.auth.outputs.token }}" \
      ${{ secrets.OAUTH4OS_URL }}/logs-*/_search \
      -d '{"query":{"match":{"level":"error"}}}'
```

## Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `proxy-url` | ✅ | — | oauth4os proxy URL |
| `client-id` | ✅ | — | OAuth client ID |
| `client-secret` | ✅ | — | OAuth client secret |
| `scope` | ❌ | `read:logs-*` | OAuth scope |
| `verify-ssl` | ❌ | `true` | Verify SSL certificates |

## Outputs

| Output | Description |
|--------|-------------|
| `token` | OAuth access token (automatically masked in logs) |
| `expires-in` | Token expiry in seconds |
| `token-type` | Token type (Bearer) |

## Examples

### Query logs after deployment

```yaml
name: Post-Deploy Smoke Test
on:
  workflow_run:
    workflows: [Deploy]
    types: [completed]

jobs:
  smoke-test:
    runs-on: ubuntu-latest
    steps:
      - uses: seraphjiang/oauth4os/action@main
        id: auth
        with:
          proxy-url: ${{ secrets.OAUTH4OS_URL }}
          client-id: ci-smoke-test
          client-secret: ${{ secrets.CI_SECRET }}
          scope: 'read:logs-*'

      - name: Check for errors in last 5 minutes
        run: |
          ERRORS=$(curl -s -H "Authorization: Bearer ${{ steps.auth.outputs.token }}" \
            "${{ secrets.OAUTH4OS_URL }}/logs-*/_count" \
            -d '{"query":{"bool":{"must":[{"match":{"level":"error"}},{"range":{"@timestamp":{"gte":"now-5m"}}}]}}}' \
            | jq .count)
          echo "Errors in last 5m: $ERRORS"
          if [ "$ERRORS" -gt 10 ]; then
            echo "::error::Too many errors after deploy: $ERRORS"
            exit 1
          fi
```

### Index test results

```yaml
- uses: seraphjiang/oauth4os/action@main
  id: auth
  with:
    proxy-url: ${{ secrets.OAUTH4OS_URL }}
    client-id: ci-writer
    client-secret: ${{ secrets.CI_WRITER_SECRET }}
    scope: 'write:ci-results-*'

- name: Index test results
  run: |
    curl -X POST -H "Authorization: Bearer ${{ steps.auth.outputs.token }}" \
      "${{ secrets.OAUTH4OS_URL }}/ci-results-${{ github.repository_owner }}/_doc" \
      -H 'Content-Type: application/json' \
      -d '{
        "repo": "${{ github.repository }}",
        "sha": "${{ github.sha }}",
        "status": "passed",
        "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
      }'
```

## Security

- The token is automatically masked in GitHub Actions logs via `::add-mask::`
- Store `client-secret` in GitHub Secrets — never hardcode
- Use the minimum scope needed for your workflow
- Tokens expire (default 1 hour) — no long-lived credentials in CI
