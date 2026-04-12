#!/usr/bin/env bash
# oauth4os-demo — CLI wrapper for oauth4os proxy
# Install: curl -sL https://f5cmk2hxwx.us-west-2.awsapprunner.com/install.sh | bash
set -euo pipefail

PROXY="${OAUTH4OS_PROXY:-https://f5cmk2hxwx.us-west-2.awsapprunner.com}"
TOKEN_FILE="${HOME}/.oauth4os/token"
CLIENT_ID="demo-cli"
REDIRECT_PORT=8199
REDIRECT_URI="http://localhost:${REDIRECT_PORT}/callback"

# Colors
RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

usage() {
  cat <<EOF
${BOLD}oauth4os-demo${NC} — OAuth 2.0 proxy CLI

${BOLD}USAGE:${NC}
  oauth4os-demo <command> [args]

${BOLD}COMMANDS:${NC}
  login              Authenticate via browser (PKCE flow)
  logout             Clear cached token
  search <query>     Search logs (e.g. 'level:ERROR')
  tail [service]     Live tail — poll every 2s, show new entries
  services           List indexed services
  indices            List OpenSearch indices
  token              Show current access token
  whoami             Show token info
  status             Check proxy health

${BOLD}ENVIRONMENT:${NC}
  OAUTH4OS_PROXY     Proxy URL (default: ${PROXY})

${BOLD}EXAMPLES:${NC}
  oauth4os-demo login
  oauth4os-demo search 'level:ERROR'
  oauth4os-demo search 'service:payment AND level:WARN'
  oauth4os-demo services
EOF
  exit 0
}

ensure_deps() {
  for cmd in curl jq; do
    command -v "$cmd" >/dev/null || { echo -e "${RED}Error: $cmd is required${NC}"; exit 1; }
  done
}

get_token() {
  if [ -f "$TOKEN_FILE" ]; then
    cat "$TOKEN_FILE"
  else
    echo -e "${RED}Not logged in. Run: oauth4os-demo login${NC}" >&2
    exit 1
  fi
}

auth_header() {
  echo "Authorization: Bearer $(get_token)"
}

# PKCE login — opens browser, starts local callback server
cmd_login() {
  mkdir -p "$(dirname "$TOKEN_FILE")"

  # Generate PKCE verifier + challenge
  CODE_VERIFIER=$(head -c 32 /dev/urandom | base64 | tr -d '=+/' | head -c 43)
  CODE_CHALLENGE=$(printf '%s' "$CODE_VERIFIER" | openssl dgst -sha256 -binary | base64 | tr '+/' '-_' | tr -d '=')
  STATE=$(head -c 16 /dev/urandom | base64 | tr -d '=+/' | head -c 16)

  AUTH_URL="${PROXY}/oauth/authorize?response_type=code&client_id=${CLIENT_ID}&redirect_uri=${REDIRECT_URI}&code_challenge=${CODE_CHALLENGE}&code_challenge_method=S256&scope=read:logs&state=${STATE}"

  echo -e "${CYAN}Opening browser for login...${NC}"
  echo -e "If browser doesn't open, visit:\n${AUTH_URL}\n"

  # Open browser
  if command -v xdg-open >/dev/null 2>&1; then xdg-open "$AUTH_URL" 2>/dev/null
  elif command -v open >/dev/null 2>&1; then open "$AUTH_URL"
  fi

  # Start temporary HTTP server to catch the callback
  echo -e "${CYAN}Waiting for callback on port ${REDIRECT_PORT}...${NC}"

  # Use a named pipe to capture the auth code
  FIFO=$(mktemp -u)
  mkfifo "$FIFO"

  # Serve one request, extract code from query string
  {
    read -r REQUEST_LINE < /dev/stdin
    CODE=$(echo "$REQUEST_LINE" | grep -oP 'code=\K[^& ]+' || true)
    echo "$CODE" > "$FIFO"
    printf 'HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nConnection: close\r\n\r\n<html><body><h2>✅ Login successful!</h2><p>You can close this tab.</p><script>window.close()</script></body></html>\r\n'
  } | nc -l -p "$REDIRECT_PORT" -q 1 2>/dev/null || \
  {
    # macOS nc syntax
    read -r REQUEST_LINE < /dev/stdin
    CODE=$(echo "$REQUEST_LINE" | grep -oE 'code=[^& ]+' | cut -d= -f2 || true)
    echo "$CODE" > "$FIFO"
    printf 'HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nConnection: close\r\n\r\n<html><body><h2>✅ Login successful!</h2><p>You can close this tab.</p></body></html>\r\n'
  } | nc -l localhost "$REDIRECT_PORT" 2>/dev/null &
  NC_PID=$!

  AUTH_CODE=$(cat "$FIFO")
  rm -f "$FIFO"
  wait $NC_PID 2>/dev/null || true

  if [ -z "${AUTH_CODE:-}" ]; then
    echo -e "${RED}Failed to get authorization code${NC}"
    exit 1
  fi

  # Exchange code for token
  RESPONSE=$(curl -sf -X POST "${PROXY}/oauth/authorize/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "grant_type=authorization_code&code=${AUTH_CODE}&client_id=${CLIENT_ID}&redirect_uri=${REDIRECT_URI}&code_verifier=${CODE_VERIFIER}")

  ACCESS_TOKEN=$(echo "$RESPONSE" | jq -r '.access_token // empty')
  if [ -z "$ACCESS_TOKEN" ]; then
    echo -e "${RED}Token exchange failed: ${RESPONSE}${NC}"
    exit 1
  fi

  echo "$ACCESS_TOKEN" > "$TOKEN_FILE"
  chmod 600 "$TOKEN_FILE"
  echo -e "${GREEN}✅ Logged in successfully${NC}"
  echo -e "Token saved to ${TOKEN_FILE}"
}

cmd_logout() {
  rm -f "$TOKEN_FILE"
  echo -e "${GREEN}Logged out${NC}"
}

cmd_search() {
  local query="${1:-*}"
  curl -sf -H "$(auth_header)" \
    "${PROXY}/demo-logs/_search" \
    -H "Content-Type: application/json" \
    -d "{\"query\":{\"query_string\":{\"query\":\"${query}\"}},\"size\":20,\"sort\":[{\"@timestamp\":{\"order\":\"desc\"}}]}" \
    | jq -r '.hits.hits[]._source | "\(.["@timestamp"]) [\(.level)] \(.service): \(.message)"' 2>/dev/null \
    || echo -e "${RED}Search failed. Are you logged in?${NC}"
}

cmd_services() {
  curl -sf -H "$(auth_header)" \
    "${PROXY}/demo-logs/_search" \
    -H "Content-Type: application/json" \
    -d '{"size":0,"aggs":{"services":{"terms":{"field":"service.keyword","size":50}}}}' \
    | jq -r '.aggregations.services.buckets[] | "\(.key) (\(.doc_count) logs)"' 2>/dev/null \
    || echo -e "${RED}Failed to list services${NC}"
}

cmd_indices() {
  curl -sf -H "$(auth_header)" "${PROXY}/_cat/indices?format=json" \
    | jq -r '.[] | "\(.index)\t\(.["docs.count"]) docs\t\(.["store.size"])"' 2>/dev/null \
    || echo -e "${RED}Failed to list indices${NC}"
}

cmd_token() {
  if [ -f "$TOKEN_FILE" ]; then
    cat "$TOKEN_FILE"
  else
    echo -e "${RED}Not logged in${NC}"
    exit 1
  fi
}

cmd_whoami() {
  local tok
  tok=$(get_token)
  # Decode JWT payload (no verification — just display)
  echo "$tok" | cut -d. -f2 | base64 -d 2>/dev/null | jq . 2>/dev/null \
    || echo "Token: ${tok:0:20}..."
}

cmd_status() {
  local resp
  resp=$(curl -sf "${PROXY}/health" 2>/dev/null)
  if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Proxy is healthy${NC}"
    echo "$resp" | jq . 2>/dev/null || echo "$resp"
  else
    echo -e "${RED}❌ Proxy unreachable${NC}"
    exit 1
  fi
}

cmd_tail() {
  local service="${1:-}" last_ts="" first=true
  echo -e "${CYAN}Live tail${service:+ for $service}${NC} (Ctrl+C to stop)\n"
  while true; do
    local filter='{"match_all":{}}'
    if [ -n "$service" ]; then
      filter="{\"term\":{\"service.keyword\":\"$service\"}}"
    fi
    local query="{\"query\":$filter,\"size\":20,\"sort\":[{\"@timestamp\":\"desc\"}]}"
    local resp
    resp=$(curl -sf -H "$(auth_header)" -H "Content-Type: application/json" \
      "${PROXY}/logs-*/_search" -d "$query" 2>/dev/null) || { sleep 2; continue; }
    local lines
    lines=$(echo "$resp" | jq -r '.hits.hits[]._source | "\(.["@timestamp"] // .timestamp // "—") [\(.level // "INFO")] \(.service // "?"): \(.message // .msg // "")"' 2>/dev/null | tac)
    if [ -n "$lines" ]; then
      local new_lines=""
      if [ -n "$last_ts" ] && ! $first; then
        new_lines=$(echo "$lines" | awk -v ts="$last_ts" '$1 > ts')
      else
        new_lines="$lines"
        first=false
      fi
      if [ -n "$new_lines" ]; then
        echo "$new_lines" | while IFS= read -r line; do
          if echo "$line" | grep -qi 'error\|fatal'; then echo -e "${RED}${line}${NC}"
          elif echo "$line" | grep -qi 'warn'; then echo -e "${YELLOW}${line}${NC}"
          else echo "$line"; fi
        done
        last_ts=$(echo "$lines" | tail -1 | awk '{print $1}')
      fi
    fi
    sleep 2
  done
}

# Main
ensure_deps
case "${1:-}" in
  login)    cmd_login ;;
  logout)   cmd_logout ;;
  search)   shift; cmd_search "$*" ;;
  services) cmd_services ;;
  indices)  cmd_indices ;;
  tail)     shift; cmd_tail "${1:-}" ;;
  token)    cmd_token ;;
  whoami)   cmd_whoami ;;
  status)   cmd_status ;;
  help|-h|--help) usage ;;
  "") usage ;;
  *) echo -e "${RED}Unknown command: $1${NC}"; usage ;;
esac
