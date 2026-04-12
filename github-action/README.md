# oauth4os GitHub Action

Get a scoped OpenSearch access token in your CI pipeline.

## Usage

```yaml
- name: Get OpenSearch token
  id: oauth
  uses: seraphjiang/oauth4os/github-action@main
  with:
    server: https://proxy.example.com
    client-id: ${{ secrets.OAUTH4OS_CLIENT_ID }}
    client-secret: ${{ secrets.OAUTH4OS_CLIENT_SECRET }}
    scope: 'read:logs-* write:metrics-*'

- name: Query OpenSearch
  run: |
    curl -H "Authorization: Bearer ${{ steps.oauth.outputs.token }}" \
      https://proxy.example.com/logs-*/_search \
      -d '{"query": {"match": {"level": "error"}}}'
```

## Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `server` | Yes | — | oauth4os proxy URL |
| `client-id` | Yes | — | OAuth client ID |
| `client-secret` | Yes | — | OAuth client secret |
| `scope` | No | `admin` | Requested scope |

## Outputs

| Output | Description |
|--------|-------------|
| `token` | Access token (masked in logs) |
| `expires-at` | Token expiry (ISO 8601) |

## Full Example

```yaml
name: Deploy and verify
on: push

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: seraphjiang/oauth4os/github-action@main
        id: oauth
        with:
          server: ${{ vars.OAUTH4OS_SERVER }}
          client-id: ${{ secrets.OAUTH4OS_CLIENT_ID }}
          client-secret: ${{ secrets.OAUTH4OS_CLIENT_SECRET }}
          scope: 'read:logs-app write:logs-app'

      - name: Index deploy marker
        run: |
          curl -X POST \
            -H "Authorization: Bearer ${{ steps.oauth.outputs.token }}" \
            -H "Content-Type: application/json" \
            "${{ vars.OAUTH4OS_SERVER }}/logs-app/_doc" \
            -d '{"event": "deploy", "sha": "${{ github.sha }}", "timestamp": "'$(date -u +%FT%TZ)'"}'

      - name: Verify no errors
        run: |
          RESULT=$(curl -s \
            -H "Authorization: Bearer ${{ steps.oauth.outputs.token }}" \
            "${{ vars.OAUTH4OS_SERVER }}/logs-app/_search" \
            -d '{"query": {"bool": {"must": [{"match": {"level": "error"}}, {"range": {"@timestamp": {"gte": "now-5m"}}}]}}}')
          COUNT=$(echo "$RESULT" | jq '.hits.total.value')
          echo "Errors in last 5m: $COUNT"
          [ "$COUNT" -eq 0 ] || exit 1
```
