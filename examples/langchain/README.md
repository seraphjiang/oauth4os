# LangChain Agent → oauth4os → OpenSearch

AI agent that queries OpenSearch logs with scoped OAuth access. The agent can search logs but cannot modify indices.

## Architecture

```
┌────────────┐     ┌──────────────┐     ┌─────────────────┐
│ LangChain  │────▶│  oauth4os    │────▶│   OpenSearch     │
│ Agent      │     │  proxy       │     │   (logs)         │
│ (read-only)│     │  read:logs-* │     │                  │
└────────────┘     └──────────────┘     └─────────────────┘
```

## Setup

```bash
pip install langchain opensearch-py requests
```

## Usage

```bash
# Get a read-only token
export OAUTH4OS_URL=http://localhost:8443
export OAUTH4OS_TOKEN=$(curl -s -X POST $OAUTH4OS_URL/oauth/token \
  -d "grant_type=client_credentials&client_id=ai-agent&client_secret=secret&scope=read:logs-*" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

# Run the agent
python3 agent.py "Find all errors in the last hour"
```

## Scope Enforcement

The agent's `read:logs-*` token can only search — any write attempt returns 403:

```bash
# This works (read)
curl -H "Authorization: Bearer $TOKEN" $OAUTH4OS_URL/logs-app/_search

# This fails (write denied by scope)
curl -X PUT -H "Authorization: Bearer $TOKEN" $OAUTH4OS_URL/logs-app/_doc/1 -d '{}'
# → 403 Forbidden
```
