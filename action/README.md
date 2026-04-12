# oauth4os GitHub Action

Get a scoped OAuth token from oauth4os in your CI pipeline.

## Usage

```yaml
- uses: seraphjiang/oauth4os/action@main
  id: auth
  with:
    proxy-url: https://oauth4os.example.com
    client-id: ${{ secrets.OAUTH4OS_CLIENT_ID }}
    client-secret: ${{ secrets.OAUTH4OS_CLIENT_SECRET }}
    scope: "read:logs-*"

- name: Query OpenSearch
  run: |
    curl -H "Authorization: Bearer ${{ steps.auth.outputs.token }}" \
      https://oauth4os.example.com/logs-*/_search \
      -d '{"query":{"match":{"level":"error"}}}'
```

## Inputs

| Input | Required | Description |
|-------|----------|-------------|
| `proxy-url` | ✅ | oauth4os proxy URL |
| `client-id` | ✅ | Client ID |
| `client-secret` | ✅ | Client secret |
| `scope` | ✅ | Requested scope |

## Outputs

| Output | Description |
|--------|-------------|
| `token` | Access token (masked in logs) |
| `expires-in` | TTL in seconds |
