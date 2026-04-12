# oauth4os GitHub Action

Get a scoped OAuth token from oauth4os for OpenSearch operations in CI/CD pipelines.

## Usage

```yaml
- name: Get OpenSearch token
  id: auth
  uses: seraphjiang/oauth4os/action@main
  with:
    proxy-url: ${{ secrets.OAUTH4OS_URL }}
    client-id: ${{ secrets.OAUTH4OS_CLIENT_ID }}
    client-secret: ${{ secrets.OAUTH4OS_CLIENT_SECRET }}
    scope: 'read:logs-* write:logs-*'

- name: Query OpenSearch
  run: |
    curl -H "Authorization: Bearer ${{ steps.auth.outputs.token }}" \
      "${{ secrets.OAUTH4OS_URL }}/logs-*/_search" \
      -d '{"query":{"match_all":{}}}'
```

## Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `proxy-url` | ✅ | — | oauth4os proxy URL |
| `client-id` | ✅ | — | OAuth client ID |
| `client-secret` | ✅ | — | OAuth client secret |
| `scope` | ❌ | `read:*` | Requested scope |

## Outputs

| Output | Description |
|--------|-------------|
| `token` | OAuth access token (masked in logs) |
| `expires-in` | Token TTL in seconds |

## Examples

### Index logs after build

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: seraphjiang/oauth4os/action@main
        id: auth
        with:
          proxy-url: ${{ secrets.OAUTH4OS_URL }}
          client-id: ci-agent
          client-secret: ${{ secrets.CI_SECRET }}
          scope: 'write:logs-*'

      - name: Push build logs
        run: |
          curl -X POST \
            -H "Authorization: Bearer ${{ steps.auth.outputs.token }}" \
            "${{ secrets.OAUTH4OS_URL }}/logs-ci/_doc" \
            -H "Content-Type: application/json" \
            -d '{"event":"deploy","status":"success","sha":"${{ github.sha }}"}'
```

### Read-only query in PR check

```yaml
      - uses: seraphjiang/oauth4os/action@main
        id: auth
        with:
          proxy-url: ${{ secrets.OAUTH4OS_URL }}
          client-id: pr-checker
          client-secret: ${{ secrets.PR_SECRET }}
          scope: 'read:logs-*'

      - name: Check for errors
        run: |
          ERRORS=$(curl -sf \
            -H "Authorization: Bearer ${{ steps.auth.outputs.token }}" \
            "${{ secrets.OAUTH4OS_URL }}/logs-*/_count" \
            -d '{"query":{"match":{"level":"error"}}}' | jq '.count')
          echo "Found $ERRORS errors"
```
