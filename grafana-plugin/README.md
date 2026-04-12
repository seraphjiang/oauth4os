# oauth4os Grafana Datasource Plugin

Query OpenSearch through oauth4os with scoped OAuth tokens — directly from Grafana.

## Features

- **Search**: Run OpenSearch queries with JSON body
- **Count**: Get document counts per index
- **Cat Indices**: List all indices
- **Scoped access**: Uses oauth4os token for auth — respects scope and Cedar policies
- **Proxy route**: Grafana backend proxies requests (no CORS issues)

## Installation

```bash
# Copy to Grafana plugins directory
cp -r grafana-plugin /var/lib/grafana/plugins/oauth4os-opensearch-datasource

# Or mount in Docker
docker run -v $(pwd)/grafana-plugin:/var/lib/grafana/plugins/oauth4os-opensearch-datasource grafana/grafana
```

## Configuration

1. Go to Grafana → Configuration → Data Sources → Add
2. Search for "oauth4os"
3. Set:
   - **Proxy URL**: Your oauth4os proxy (e.g., `http://localhost:8443`)
   - **Access Token**: OAuth token with `read:*` scope

## Query Examples

### Search
```json
{
  "query": {
    "bool": {
      "must": [
        { "match": { "level": "error" } },
        { "range": { "@timestamp": { "gte": "now-1h" } } }
      ]
    }
  },
  "size": 100
}
```

### Count
```json
{ "query": { "match": { "level": "error" } } }
```

## Development

```bash
cd grafana-plugin
npm install
npm run dev    # watch mode
npm run build  # production build
```
