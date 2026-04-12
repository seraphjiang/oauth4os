#!/usr/bin/env bash
# oauth4os-demo — CLI wrapper for oauth4os proxy
# Install: curl -sL https://f5cmk2hxwx.us-west-2.awsapprunner.com/install.sh | bash
set -euo pipefail

CONFIG_FILE="${HOME}/.oauth4os/config"
TOKEN_FILE="${HOME}/.oauth4os/token"
ALIAS_FILE="${HOME}/.oauth4os/aliases"

# Load config defaults, then override from config file
_default_proxy="https://f5cmk2hxwx.us-west-2.awsapprunner.com"
_default_index="logs-*"
_default_format="text"
if [ -f "$CONFIG_FILE" ]; then
  _cfg_proxy=$(grep '^proxy=' "$CONFIG_FILE" 2>/dev/null | cut -d= -f2-)
  _cfg_index=$(grep '^index=' "$CONFIG_FILE" 2>/dev/null | cut -d= -f2-)
  _cfg_format=$(grep '^format=' "$CONFIG_FILE" 2>/dev/null | cut -d= -f2-)
fi
PROXY="${OAUTH4OS_PROXY:-${_cfg_proxy:-$_default_proxy}}"
DEFAULT_INDEX="${OAUTH4OS_INDEX:-${_cfg_index:-$_default_index}}"
DEFAULT_FORMAT="${OAUTH4OS_FORMAT:-${_cfg_format:-$_default_format}}"
CLIENT_ID="demo-cli"
REDIRECT_PORT=8199
REDIRECT_URI="http://localhost:${REDIRECT_PORT}/callback"

# --json flag: force machine-readable JSON output
JSON_MODE=false
for arg in "$@"; do
  [ "$arg" = "--json" ] && JSON_MODE=true
done

# Colors
RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; YELLOW='\033[1;33m'; BOLD='\033[1m'; NC='\033[0m'

# Disable colors and output raw JSON when piped or --json
IS_TTY=true
if [ ! -t 1 ] || $JSON_MODE; then
  IS_TTY=false
  RED=''; GREEN=''; CYAN=''; YELLOW=''; BOLD=''; NC=''
fi

usage() {
  cat <<EOF
${BOLD}oauth4os-demo${NC} — OAuth 2.0 proxy CLI

${BOLD}USAGE:${NC}
  oauth4os-demo [--json] <command> [args]

${BOLD}FLAGS:${NC}
  --json             Force JSON output (machine-readable)

${BOLD}COMMANDS:${NC}
  login              Authenticate via browser (PKCE flow)
  logout             Clear cached token
  search <query>     Search logs with KQL (e.g. 'level:ERROR')
  tail [service]     Live tail — poll every 2s, show new entries
  services           List indexed services
  indices            List OpenSearch indices
  token              Show current access token
  whoami             Show token info
  status             Check proxy health
  stats              Error counts, latency, top errors by service
  export <q> -f csv|json -o <file>  Export results to file
  sql <query>          Run SQL query against OpenSearch
  history              Show last 50 queries
  bookmark <action>    save|run|delete|list query bookmarks
  dashboard            Live terminal dashboard (htop for logs)
  watch <query>        Alert on new KQL matches (poll every 5s)
  diff <r1> <r2>      Compare time ranges (today, yesterday, 1h, 24h, 7d)
  config <action>      show|set|get|reset proxy settings
  alias <action>       add|rm|run|list command aliases
  completion <shell>   Generate bash/zsh completions
  profile              Formatted token claims, scopes, expiry
  top                  Real-time top consumers (like Unix top)
  env                  Show config, paths, connectivity diagnostic
  audit [n]            Show last n admin audit events (default 20)
  alerts               Show alert status from proxy metrics
  latency              Show upstream latency and throughput
  ping [n]             Measure round-trip latency (default 5 pings)
  install-man          Install man page to system

${BOLD}ENVIRONMENT:${NC}
  OAUTH4OS_PROXY     Proxy URL (default: ${PROXY})

${BOLD}KQL SYNTAX:${NC}
  field:value              Exact match (service:payment)
  field:>N / field:<N      Range (latency_ms:>500)
  field:>=N / field:<=N    Range inclusive
  field:val*               Wildcard (service:pay*)
  term1 AND term2          Both must match
  term1 OR term2           Either matches
  NOT term                 Exclude

${BOLD}EXAMPLES:${NC}
  oauth4os-demo login
  oauth4os-demo search 'level:ERROR'
  oauth4os-demo search 'service:payment AND level:WARN'
  oauth4os-demo search 'latency_ms:>500'
  oauth4os-demo search 'service:auth* AND NOT level:INFO'
  oauth4os-demo tail payment
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

kql_to_dsl() {
  # Convert KQL-style query to OpenSearch query DSL JSON
  # Supports: field:value, field:>N, field:<N, field:>=N, field:<=N, AND, OR, NOT, wildcards
  local input="$1"

  # Pass-through for simple wildcard or empty
  if [ "$input" = "*" ] || [ -z "$input" ]; then
    echo '{"match_all":{}}'
    return
  fi

  # Split on AND/OR and build bool query
  # For simple single-term KQL like "service:payment" or "level:ERROR"
  local clauses=() op="must"
  local remaining="$input"

  # Normalize: ensure spaces around AND/OR/NOT
  remaining=$(echo "$remaining" | sed 's/  */ /g')

  # Build using jq for safe JSON construction
  local must_clauses=() must_not_clauses=() should_clauses=()
  local use_should=false negate=false

  for token in $remaining; do
    case "$token" in
      AND) op="must"; continue ;;
      OR)  use_should=true; op="should"; continue ;;
      NOT) negate=true; continue ;;
    esac

    local clause=""
    if echo "$token" | grep -q ':>='; then
      local field="${token%%:>=*}" val="${token#*:>=}"
      clause="{\"range\":{\"${field}\":{\"gte\":${val}}}}"
    elif echo "$token" | grep -q ':<='; then
      local field="${token%%:<=*}" val="${token#*:<=}"
      clause="{\"range\":{\"${field}\":{\"lte\":${val}}}}"
    elif echo "$token" | grep -q ':>'; then
      local field="${token%%:>*}" val="${token#*:>}"
      clause="{\"range\":{\"${field}\":{\"gt\":${val}}}}"
    elif echo "$token" | grep -q ':<'; then
      local field="${token%%:<*}" val="${token#*:<}"
      clause="{\"range\":{\"${field}\":{\"lt\":${val}}}}"
    elif echo "$token" | grep -q ':'; then
      local field="${token%%:*}" val="${token#*:}"
      if echo "$val" | grep -q '[*?]'; then
        clause="{\"wildcard\":{\"${field}.keyword\":{\"value\":\"${val}\"}}}"
      else
        clause="{\"match\":{\"${field}\":\"${val}\"}}"
      fi
    else
      clause="{\"multi_match\":{\"query\":\"${token}\",\"fields\":[\"message\",\"service\",\"level\"]}}"
    fi

    if $negate; then
      must_not_clauses+=("$clause")
      negate=false
    elif [ "$op" = "should" ]; then
      should_clauses+=("$clause")
    else
      must_clauses+=("$clause")
    fi
  done

  # Assemble bool query
  local parts=()
  if [ ${#must_clauses[@]} -gt 0 ]; then
    local joined=$(IFS=,; echo "${must_clauses[*]}")
    parts+=("\"must\":[${joined}]")
  fi
  if [ ${#must_not_clauses[@]} -gt 0 ]; then
    local joined=$(IFS=,; echo "${must_not_clauses[*]}")
    parts+=("\"must_not\":[${joined}]")
  fi
  if [ ${#should_clauses[@]} -gt 0 ]; then
    local joined=$(IFS=,; echo "${should_clauses[*]}")
    parts+=("\"should\":[${joined}],\"minimum_should_match\":1")
  fi

  if [ ${#parts[@]} -eq 0 ]; then
    echo '{"match_all":{}}'
  elif [ ${#must_clauses[@]} -eq 1 ] && [ ${#must_not_clauses[@]} -eq 0 ] && [ ${#should_clauses[@]} -eq 0 ]; then
    echo "${must_clauses[0]}"
  else
    local joined=$(IFS=,; echo "${parts[*]}")
    echo "{\"bool\":{${joined}}}"
  fi
}

cmd_search() {
  local query="${1:-*}"
  local dsl
  dsl=$(kql_to_dsl "$query")
  save_history "$query"
  local body="{\"query\":${dsl},\"size\":20,\"sort\":[{\"@timestamp\":{\"order\":\"desc\"}}]}"
  local resp
  resp=$(curl -sf -H "$(auth_header)" \
    "${PROXY}/${DEFAULT_INDEX}/_search" \
    -H "Content-Type: application/json" \
    -d "$body" 2>/dev/null)
  if [ $? -ne 0 ] || [ -z "$resp" ]; then
    echo -e "${RED}Search failed. Are you logged in?${NC}" >&2
    return 1
  fi
  # Pipe mode: output raw JSON array of _source docs
  if ! $IS_TTY; then
    echo "$resp" | jq '[.hits.hits[]._source]' 2>/dev/null
    return
  fi
  echo -e "${CYAN}Query:${NC} $query"
  local total
  total=$(echo "$resp" | jq '.hits.total.value // (.hits.total // 0)' 2>/dev/null)
  echo -e "${GREEN}${total} hits${NC}\n"
  echo "$resp" | jq -r '.hits.hits[]._source | "\(.["@timestamp"] // .timestamp // "—") [\(.level // "INFO")] \(.service // "?"): \(.message // .msg // "")"' 2>/dev/null | while IFS= read -r line; do
    if echo "$line" | grep -qi 'error\|fatal'; then echo -e "${RED}${line}${NC}"
    elif echo "$line" | grep -qi 'warn'; then echo -e "${YELLOW}${line}${NC}"
    else echo "$line"; fi
  done
}

cmd_services() {
  local resp
  resp=$(curl -sf -H "$(auth_header)" \
    "${PROXY}/demo-logs/_search" \
    -H "Content-Type: application/json" \
    -d '{"size":0,"aggs":{"services":{"terms":{"field":"service.keyword","size":50}}}}' 2>/dev/null)
  if [ -z "$resp" ]; then echo -e "${RED}Failed to list services${NC}"; return 1; fi
  if [ "$IS_TTY" = "false" ]; then echo "$resp" | jq '.aggregations.services.buckets' 2>/dev/null || echo "$resp"; return; fi
  echo "$resp" | jq -r '.aggregations.services.buckets[] | "\(.key) (\(.doc_count) logs)"' 2>/dev/null
}

cmd_indices() {
  local resp
  resp=$(curl -sf -H "$(auth_header)" "${PROXY}/_cat/indices?format=json" 2>/dev/null)
  if [ -z "$resp" ]; then echo -e "${RED}Failed to list indices${NC}"; return 1; fi
  if [ "$IS_TTY" = "false" ]; then echo "$resp"; return; fi
  echo "$resp" | jq -r '.[] | "\(.index)\t\(.["docs.count"]) docs\t\(.["store.size"])"' 2>/dev/null
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
    if [ "$IS_TTY" = "false" ]; then echo "$resp"; return; fi
    echo -e "${GREEN}✅ Proxy is healthy${NC}"
    echo "$resp" | jq . 2>/dev/null || echo "$resp"
  else
    if [ "$IS_TTY" = "false" ]; then echo '{"status":"unreachable"}'; exit 1; fi
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
      "${PROXY}/${DEFAULT_INDEX}/_search" -d "$query" 2>/dev/null) || { sleep 2; continue; }
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

cmd_stats() {
  local tok
  tok=$(get_token) || { echo -e "${RED}Not logged in. Run: oauth4os-demo login${NC}"; return 1; }
  echo -e "${BOLD}📊 Index Stats${NC}\n"

  # Single multi-agg query
  local body='{
    "size":0,
    "aggs":{
      "by_service":{"terms":{"field":"service.keyword","size":20},"aggs":{
        "errors":{"filter":{"terms":{"level.keyword":["ERROR","FATAL"]}}},
        "avg_latency":{"avg":{"field":"latency_ms"}}
      }},
      "top_errors":{"filter":{"terms":{"level.keyword":["ERROR","FATAL"]}},"aggs":{
        "messages":{"terms":{"field":"message.keyword","size":10}}
      }},
      "total_errors":{"filter":{"terms":{"level.keyword":["ERROR","FATAL"]}}},
      "total_docs":{"value_count":{"field":"_index"}}
    }
  }'
  local resp
  resp=$(curl -sf -H "Authorization: Bearer ${tok}" -H "Content-Type: application/json" \
    "${PROXY}/${DEFAULT_INDEX}/_search" -d "$body" 2>/dev/null)
  if [ $? -ne 0 ] || [ -z "$resp" ]; then
    echo -e "${RED}Query failed${NC}"; return 1
  fi

  if [ "$IS_TTY" = "false" ]; then echo "$resp" | jq '.aggregations' 2>/dev/null || echo "$resp"; return; fi

  # Summary
  local total errs
  total=$(echo "$resp" | jq '.hits.total.value // (.hits.total // 0)' 2>/dev/null)
  errs=$(echo "$resp" | jq '.aggregations.total_errors.doc_count // 0' 2>/dev/null)
  echo -e "  Total docs: ${CYAN}${total}${NC}    Errors: ${RED}${errs}${NC}\n"

  # Errors by service
  echo -e "${BOLD}Errors by Service:${NC}"
  printf "  ${CYAN}%-20s %8s %8s %12s${NC}\n" "SERVICE" "TOTAL" "ERRORS" "AVG LATENCY"
  printf "  %-20s %8s %8s %12s\n" "────────────────────" "────────" "────────" "────────────"
  echo "$resp" | jq -r '.aggregations.by_service.buckets[] | "\(.key) \(.doc_count) \(.errors.doc_count) \(.avg_latency.value // 0)"' 2>/dev/null | while read -r svc total err lat; do
    lat_fmt=$(printf "%.1fms" "$lat" 2>/dev/null || echo "${lat}ms")
    if [ "$err" -gt 0 ] 2>/dev/null; then
      printf "  %-20s %8s ${RED}%8s${NC} %12s\n" "$svc" "$total" "$err" "$lat_fmt"
    else
      printf "  %-20s %8s ${GREEN}%8s${NC} %12s\n" "$svc" "$total" "$err" "$lat_fmt"
    fi
  done

  # Top error messages
  echo ""
  echo -e "${BOLD}Top Error Messages:${NC}"
  echo "$resp" | jq -r '.aggregations.top_errors.messages.buckets[:10][] | "\(.doc_count) \(.key)"' 2>/dev/null | while read -r cnt msg; do
    printf "  ${RED}%5s${NC}  %s\n" "$cnt" "$msg"
  done

  local no_errs
  no_errs=$(echo "$resp" | jq '.aggregations.top_errors.messages.buckets | length' 2>/dev/null)
  if [ "${no_errs:-0}" = "0" ]; then
    echo -e "  ${GREEN}No errors found ✓${NC}"
  fi
}

cmd_export() {
  local query="" fmt="json" outfile=""
  while [ $# -gt 0 ]; do
    case "$1" in
      --format|-f) fmt="$2"; shift 2 ;;
      --output|-o) outfile="$2"; shift 2 ;;
      *) query="${query:+$query }$1"; shift ;;
    esac
  done
  query="${query:-*}"
  [ -z "$outfile" ] && { echo -e "${RED}Usage: oauth4os-demo export <query> --format csv|json --output <file>${NC}"; return 1; }

  local tok
  tok=$(get_token) || { echo -e "${RED}Not logged in${NC}"; return 1; }
  local dsl
  dsl=$(kql_to_dsl "$query")
  local body="{\"query\":${dsl},\"size\":10000,\"sort\":[{\"@timestamp\":{\"order\":\"desc\"}}]}"
  echo -e "${CYAN}Exporting:${NC} $query → $outfile ($fmt)"

  local resp
  resp=$(curl -sf -H "Authorization: Bearer ${tok}" -H "Content-Type: application/json" \
    "${PROXY}/${DEFAULT_INDEX}/_search" -d "$body" 2>/dev/null)
  [ $? -ne 0 ] && { echo -e "${RED}Query failed${NC}"; return 1; }

  local count
  count=$(echo "$resp" | jq '.hits.hits | length' 2>/dev/null)

  if [ "$fmt" = "csv" ]; then
    echo "timestamp,level,service,message" > "$outfile"
    echo "$resp" | jq -r '.hits.hits[]._source | [(.["@timestamp"] // .timestamp // ""), (.level // ""), (.service // ""), (.message // .msg // "")] | @csv' >> "$outfile" 2>/dev/null
  else
    echo "$resp" | jq '[.hits.hits[]._source]' > "$outfile" 2>/dev/null
  fi

  echo -e "${GREEN}✓ Exported ${count} records to ${outfile}${NC}"
}

HISTORY_FILE="${HOME}/.oauth4os-history"
BOOKMARKS_FILE="${HOME}/.oauth4os-bookmarks"

save_history() {
  local q="$1"
  [ -z "$q" ] || [ "$q" = "*" ] && return
  # Prepend, dedup, keep 50
  local tmp
  tmp=$(mktemp)
  echo "$q" > "$tmp"
  [ -f "$HISTORY_FILE" ] && grep -vxF "$q" "$HISTORY_FILE" | head -49 >> "$tmp"
  mv "$tmp" "$HISTORY_FILE"
}

cmd_sql() {
  local sql="$1"
  [ -z "$sql" ] && { echo -e "${YELLOW}Usage: oauth4os-demo sql 'SELECT * FROM logs-demo WHERE level=\\'ERROR\\' LIMIT 10'${NC}"; return 1; }
  local tok
  tok=$(get_token) || { echo -e "${RED}Not logged in${NC}"; return 1; }
  save_history "sql: $sql"
  if $IS_TTY; then echo -e "${CYAN}SQL:${NC} $sql\n"; fi
  local body
  body=$(jq -n --arg q "$sql" '{query:$q}')
  local resp
  resp=$(curl -sf -H "Authorization: Bearer ${tok}" -H "Content-Type: application/json" \
    "${PROXY}/_plugins/_sql" -d "$body" 2>/dev/null)
  if [ $? -ne 0 ] || [ -z "$resp" ]; then
    # Fallback: try _sql endpoint without _plugins prefix
    resp=$(curl -sf -H "Authorization: Bearer ${tok}" -H "Content-Type: application/json" \
      "${PROXY}/_sql" -d "$body" 2>/dev/null)
  fi
  if [ $? -ne 0 ] || [ -z "$resp" ]; then
    echo -e "${RED}SQL query failed${NC}"; return 1
  fi
  # Check for error
  local err
  err=$(echo "$resp" | jq -r '.error.reason // empty' 2>/dev/null)
  if [ -n "$err" ]; then
    echo -e "${RED}Error: $err${NC}"; return 1
  fi
  # Format: schema + datarows (OpenSearch SQL response format)
  local has_schema
  has_schema=$(echo "$resp" | jq 'has("schema")' 2>/dev/null)
  if [ "$has_schema" = "true" ]; then
    # Print column headers
    local headers
    headers=$(echo "$resp" | jq -r '[.schema[].name] | join("\t")' 2>/dev/null)
    echo -e "${BOLD}${headers}${NC}"
    echo "$resp" | jq -r '.datarows[] | [.[] | tostring] | join("\t")' 2>/dev/null
    local rows
    rows=$(echo "$resp" | jq '.datarows | length' 2>/dev/null)
    echo -e "\n${GREEN}${rows} rows${NC}"
  else
    # Fallback: pretty print
    echo "$resp" | jq . 2>/dev/null || echo "$resp"
  fi
}

cmd_history() {
  if [ ! -f "$HISTORY_FILE" ] || [ ! -s "$HISTORY_FILE" ]; then
    echo -e "${YELLOW}No query history yet${NC}"; return
  fi
  echo -e "${BOLD}Recent Queries:${NC}\n"
  local i=1
  while IFS= read -r line; do
    printf "  ${CYAN}%3d${NC}  %s\n" "$i" "$line"
    i=$((i+1))
  done < "$HISTORY_FILE"
}

cmd_bookmark() {
  local action="${1:-}" name="${2:-}" query="${3:-}"
  case "$action" in
    save)
      [ -z "$name" ] || [ -z "$query" ] && { echo -e "${YELLOW}Usage: oauth4os-demo bookmark save <name> <query>${NC}"; return 1; }
      # Remove existing with same name, append new
      [ -f "$BOOKMARKS_FILE" ] && grep -v "^${name}	" "$BOOKMARKS_FILE" > "${BOOKMARKS_FILE}.tmp" && mv "${BOOKMARKS_FILE}.tmp" "$BOOKMARKS_FILE"
      echo -e "${name}\t${query}" >> "$BOOKMARKS_FILE"
      echo -e "${GREEN}✓ Saved bookmark '${name}'${NC}: $query"
      ;;
    run)
      [ -z "$name" ] && { echo -e "${YELLOW}Usage: oauth4os-demo bookmark run <name>${NC}"; return 1; }
      [ ! -f "$BOOKMARKS_FILE" ] && { echo -e "${RED}No bookmarks${NC}"; return 1; }
      local q
      q=$(grep "^${name}	" "$BOOKMARKS_FILE" | cut -f2-)
      [ -z "$q" ] && { echo -e "${RED}Bookmark '${name}' not found${NC}"; return 1; }
      echo -e "${CYAN}Running bookmark '${name}':${NC} $q\n"
      cmd_search "$q"
      ;;
    delete|rm)
      [ -z "$name" ] && { echo -e "${YELLOW}Usage: oauth4os-demo bookmark delete <name>${NC}"; return 1; }
      [ -f "$BOOKMARKS_FILE" ] && grep -v "^${name}	" "$BOOKMARKS_FILE" > "${BOOKMARKS_FILE}.tmp" && mv "${BOOKMARKS_FILE}.tmp" "$BOOKMARKS_FILE"
      echo -e "${GREEN}✓ Deleted bookmark '${name}'${NC}"
      ;;
    list|"")
      if [ ! -f "$BOOKMARKS_FILE" ] || [ ! -s "$BOOKMARKS_FILE" ]; then
        echo -e "${YELLOW}No bookmarks yet. Save one: oauth4os-demo bookmark save errors 'level:ERROR'${NC}"; return
      fi
      echo -e "${BOLD}Bookmarks:${NC}\n"
      while IFS=$'\t' read -r n q; do
        echo -e "  ${CYAN}${n}${NC}  →  $q"
      done < "$BOOKMARKS_FILE"
      ;;
    *) echo -e "${YELLOW}Usage: oauth4os-demo bookmark <save|run|delete|list> [name] [query]${NC}" ;;
  esac
}

cmd_dashboard() {
  local tok
  tok=$(get_token) || { echo -e "${RED}Not logged in${NC}"; return 1; }
  local prev_total=0 prev_time=$(date +%s)

  trap 'tput cnorm; echo; exit 0' INT
  tput civis  # hide cursor

  while true; do
    local now=$(date +%s)
    # Fetch stats in one multi-agg query
    local body='{
      "size":5,"sort":[{"@timestamp":"desc"}],
      "query":{"match_all":{}},
      "aggs":{
        "errors":{"filter":{"terms":{"level.keyword":["ERROR","FATAL"]}}},
        "by_service":{"terms":{"field":"service.keyword","size":10}},
        "by_level":{"terms":{"field":"level.keyword","size":5}},
        "recent_errors":{"filter":{"terms":{"level.keyword":["ERROR","FATAL"]}},"aggs":{
          "top":{"top_hits":{"size":5,"sort":[{"@timestamp":"desc"}],"_source":["@timestamp","service","message"]}}
        }}
      }
    }'
    local resp
    resp=$(curl -sf -H "Authorization: Bearer ${tok}" -H "Content-Type: application/json" \
      "${PROXY}/${DEFAULT_INDEX}/_search" -d "$body" 2>/dev/null)
    [ $? -ne 0 ] && { sleep 3; continue; }

    local total errs
    total=$(echo "$resp" | jq '.hits.total.value // 0' 2>/dev/null)
    errs=$(echo "$resp" | jq '.aggregations.errors.doc_count // 0' 2>/dev/null)

    # Request rate
    local elapsed=$(( now - prev_time ))
    local rate=0
    [ $elapsed -gt 0 ] && [ $prev_total -gt 0 ] && rate=$(( (total - prev_total) / elapsed ))
    prev_total=$total; prev_time=$now

    # Clear screen and draw
    tput clear
    local ts=$(date '+%H:%M:%S')
    echo -e "${BOLD}🔐 oauth4os Dashboard${NC}                                    ${CYAN}${ts}${NC}  (q=quit)"
    echo -e "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    # Summary row
    local err_pct=0
    [ "$total" -gt 0 ] 2>/dev/null && err_pct=$(( errs * 100 / total ))
    printf "  ${BOLD}Total Docs${NC}  %-12s" "$total"
    printf "${BOLD}Errors${NC}  ${RED}%-8s${NC}" "$errs"
    printf "${BOLD}Error Rate${NC}  "
    [ "$err_pct" -gt 10 ] && printf "${RED}%s%%${NC}" "$err_pct" || printf "${GREEN}%s%%${NC}" "$err_pct"
    printf "    ${BOLD}Rate${NC}  ${CYAN}%s/s${NC}\n" "$rate"
    echo ""

    # Level distribution bar
    echo -e "  ${BOLD}Levels:${NC}"
    echo "$resp" | jq -r '.aggregations.by_level.buckets[] | "\(.key) \(.doc_count)"' 2>/dev/null | while read -r lvl cnt; do
      local bar_len=$(( cnt * 40 / (total > 0 ? total : 1) ))
      [ $bar_len -lt 1 ] && bar_len=1
      local bar=$(printf '%*s' "$bar_len" '' | tr ' ' '█')
      case "$lvl" in
        ERROR|FATAL) printf "  ${RED}%-8s %6s ${NC}${RED}%s${NC}\n" "$lvl" "$cnt" "$bar" ;;
        WARN)        printf "  ${YELLOW}%-8s %6s ${NC}${YELLOW}%s${NC}\n" "$lvl" "$cnt" "$bar" ;;
        *)           printf "  ${GREEN}%-8s %6s ${NC}${GREEN}%s${NC}\n" "$lvl" "$cnt" "$bar" ;;
      esac
    done
    echo ""

    # Top services
    echo -e "  ${BOLD}Top Services:${NC}"
    printf "  ${CYAN}%-20s %8s${NC}\n" "SERVICE" "DOCS"
    echo "$resp" | jq -r '.aggregations.by_service.buckets[] | "\(.key) \(.doc_count)"' 2>/dev/null | while read -r svc cnt; do
      local bar_len=$(( cnt * 30 / (total > 0 ? total : 1) ))
      [ $bar_len -lt 1 ] && bar_len=1
      local bar=$(printf '%*s' "$bar_len" '' | tr ' ' '▓')
      printf "  %-20s %8s  ${CYAN}%s${NC}\n" "$svc" "$cnt" "$bar"
    done
    echo ""

    # Latest errors
    echo -e "  ${BOLD}Latest Errors:${NC}"
    local err_lines
    err_lines=$(echo "$resp" | jq -r '.aggregations.recent_errors.top.hits.hits[]._source | "\(.["@timestamp"] // "?") \(.service // "?"): \(.message // "")"' 2>/dev/null)
    if [ -n "$err_lines" ]; then
      echo "$err_lines" | head -5 | while IFS= read -r line; do
        echo -e "  ${RED}${line}${NC}"
      done
    else
      echo -e "  ${GREEN}No recent errors ✓${NC}"
    fi

    echo -e "\n  ${CYAN}Refreshing in 3s...${NC}"

    # Check for 'q' keypress (non-blocking)
    read -t 3 -n 1 key 2>/dev/null && [ "$key" = "q" ] && { tput cnorm; echo; break; }
  done
}

cmd_config() {
  mkdir -p "$(dirname "$CONFIG_FILE")"
  local action="${1:-show}" key="${2:-}" val="${3:-}"
  case "$action" in
    set)
      [ -z "$key" ] || [ -z "$val" ] && { echo -e "${YELLOW}Usage: oauth4os-demo config set <key> <value>${NC}"; echo "  Keys: proxy, index, format"; return 1; }
      case "$key" in proxy|index|format) ;; *) echo -e "${RED}Unknown key: $key (valid: proxy, index, format)${NC}"; return 1 ;; esac
      # Update or append
      if [ -f "$CONFIG_FILE" ] && grep -q "^${key}=" "$CONFIG_FILE"; then
        sed -i "s|^${key}=.*|${key}=${val}|" "$CONFIG_FILE"
      else
        echo "${key}=${val}" >> "$CONFIG_FILE"
      fi
      echo -e "${GREEN}✓ ${key}=${val}${NC}"
      ;;
    get)
      [ -z "$key" ] && { echo -e "${YELLOW}Usage: oauth4os-demo config get <key>${NC}"; return 1; }
      [ -f "$CONFIG_FILE" ] && grep "^${key}=" "$CONFIG_FILE" | cut -d= -f2- || echo "(not set)"
      ;;
    show|"")
      echo -e "${BOLD}Config:${NC} ${CONFIG_FILE}"
      echo -e "  proxy:  ${CYAN}${PROXY}${NC}"
      echo -e "  index:  ${CYAN}${DEFAULT_INDEX}${NC}"
      echo -e "  format: ${CYAN}${DEFAULT_FORMAT}${NC}"
      echo -e "\n${BOLD}Files:${NC}"
      echo -e "  token:     ${TOKEN_FILE}"
      echo -e "  history:   ${HISTORY_FILE:-~/.oauth4os-history}"
      echo -e "  bookmarks: ${BOOKMARKS_FILE:-~/.oauth4os-bookmarks}"
      echo -e "  aliases:   ${ALIAS_FILE}"
      ;;
    reset)
      rm -f "$CONFIG_FILE"
      echo -e "${GREEN}✓ Config reset to defaults${NC}"
      ;;
    *) echo -e "${YELLOW}Usage: oauth4os-demo config <show|set|get|reset> [key] [value]${NC}" ;;
  esac
}

cmd_alias() {
  mkdir -p "$(dirname "$ALIAS_FILE")"
  local action="${1:-list}" name="${2:-}" cmd="${3:-}"
  case "$action" in
    add|save)
      [ -z "$name" ] || [ -z "$cmd" ] && { echo -e "${YELLOW}Usage: oauth4os-demo alias add <name> <command>${NC}"; return 1; }
      [ -f "$ALIAS_FILE" ] && grep -v "^${name}	" "$ALIAS_FILE" > "${ALIAS_FILE}.tmp" 2>/dev/null && mv "${ALIAS_FILE}.tmp" "$ALIAS_FILE"
      echo -e "${name}\t${cmd}" >> "$ALIAS_FILE"
      echo -e "${GREEN}✓ Alias '${name}' → ${cmd}${NC}"
      ;;
    rm|delete)
      [ -z "$name" ] && { echo -e "${YELLOW}Usage: oauth4os-demo alias rm <name>${NC}"; return 1; }
      [ -f "$ALIAS_FILE" ] && grep -v "^${name}	" "$ALIAS_FILE" > "${ALIAS_FILE}.tmp" && mv "${ALIAS_FILE}.tmp" "$ALIAS_FILE"
      echo -e "${GREEN}✓ Removed alias '${name}'${NC}"
      ;;
    run)
      [ -z "$name" ] && { echo -e "${YELLOW}Usage: oauth4os-demo alias run <name>${NC}"; return 1; }
      [ ! -f "$ALIAS_FILE" ] && { echo -e "${RED}No aliases${NC}"; return 1; }
      local acmd
      acmd=$(grep "^${name}	" "$ALIAS_FILE" | cut -f2-)
      [ -z "$acmd" ] && { echo -e "${RED}Alias '${name}' not found${NC}"; return 1; }
      echo -e "${CYAN}Running alias '${name}':${NC} $acmd"
      eval "cmd_search \"$acmd\""
      ;;
    list|"")
      if [ ! -f "$ALIAS_FILE" ] || [ ! -s "$ALIAS_FILE" ]; then
        echo -e "${YELLOW}No aliases. Create one: oauth4os-demo alias add errors 'search level:ERROR'${NC}"; return
      fi
      echo -e "${BOLD}Aliases:${NC}\n"
      while IFS=$'\t' read -r n c; do
        echo -e "  ${CYAN}${n}${NC}  →  $c"
      done < "$ALIAS_FILE"
      ;;
    *) echo -e "${YELLOW}Usage: oauth4os-demo alias <add|rm|run|list> [name] [command]${NC}" ;;
  esac
}

cmd_completion() {
  local shell="${1:-bash}"
  case "$shell" in
    bash)
      cat <<'COMP'
_oauth4os_demo() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  local cmds="login logout search sql tail services indices stats export dashboard bookmark history config alias completion status token whoami help"
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=($(compgen -W "$cmds" -- "$cur"))
  elif [ "${COMP_WORDS[1]}" = "config" ] && [ "$COMP_CWORD" -eq 2 ]; then
    COMPREPLY=($(compgen -W "show set get reset" -- "$cur"))
  elif [ "${COMP_WORDS[1]}" = "config" ] && [ "${COMP_WORDS[2]}" = "set" ] && [ "$COMP_CWORD" -eq 3 ]; then
    COMPREPLY=($(compgen -W "proxy index format" -- "$cur"))
  elif [ "${COMP_WORDS[1]}" = "alias" ] && [ "$COMP_CWORD" -eq 2 ]; then
    COMPREPLY=($(compgen -W "add rm run list" -- "$cur"))
  elif [ "${COMP_WORDS[1]}" = "bookmark" ] && [ "$COMP_CWORD" -eq 2 ]; then
    COMPREPLY=($(compgen -W "save run delete list" -- "$cur"))
  elif [ "${COMP_WORDS[1]}" = "export" ]; then
    COMPREPLY=($(compgen -W "-f --format -o --output csv json" -- "$cur"))
  fi
}
complete -F _oauth4os_demo oauth4os-demo
COMP
      echo -e "\n# Add to ~/.bashrc: eval \"\$(oauth4os-demo completion bash)\"" >&2
      ;;
    zsh)
      cat <<'COMP'
#compdef oauth4os-demo
_oauth4os_demo() {
  local -a commands=(login logout search sql tail services indices stats export dashboard bookmark history config alias completion status token whoami help)
  _arguments '1:command:compadd -a commands'
}
compdef _oauth4os_demo oauth4os-demo
COMP
      echo -e "\n# Add to ~/.zshrc: eval \"\$(oauth4os-demo completion zsh)\"" >&2
      ;;
    *) echo -e "${YELLOW}Usage: oauth4os-demo completion <bash|zsh>${NC}" ;;
  esac
}

cmd_watch() {
  local query="${1:-*}" interval="${WATCH_INTERVAL:-5}" last_count=-1
  local tok
  tok=$(get_token) || { echo -e "${RED}Not logged in${NC}"; return 1; }
  local dsl
  dsl=$(kql_to_dsl "$query")
  echo -e "${BOLD}👁 Watching:${NC} $query  (every ${interval}s, Ctrl+C to stop)\n"

  trap 'echo -e "\n${NC}Stopped."; exit 0' INT
  while true; do
    local body="{\"query\":${dsl},\"size\":5,\"sort\":[{\"@timestamp\":{\"order\":\"desc\"}}]}"
    local resp
    resp=$(curl -sf -H "Authorization: Bearer ${tok}" -H "Content-Type: application/json" \
      "${PROXY}/${DEFAULT_INDEX}/_search" -d "$body" 2>/dev/null)
    if [ $? -ne 0 ]; then sleep "$interval"; continue; fi

    local count
    count=$(echo "$resp" | jq '.hits.total.value // 0' 2>/dev/null)

    if [ "$last_count" -ge 0 ] 2>/dev/null && [ "$count" -gt "$last_count" ]; then
      local new=$(( count - last_count ))
      local ts=$(date '+%H:%M:%S')
      echo -e "${RED}🔔 [${ts}] ALERT: ${new} new match(es) (${last_count}→${count})${NC}"
      echo "$resp" | jq -r '.hits.hits[:3][]._source | "   \(.["@timestamp"] // "?") [\(.level // "?")] \(.service // "?"): \(.message // "")"' 2>/dev/null | while IFS= read -r line; do
        echo -e "  ${YELLOW}${line}${NC}"
      done
      echo ""
    elif [ "$last_count" -eq -1 ]; then
      echo -e "${CYAN}Baseline: ${count} matches${NC}\n"
    fi
    last_count=$count
    sleep "$interval"
  done
}

cmd_diff() {
  local range1="${1:-today}" range2="${2:-yesterday}"
  local tok
  tok=$(get_token) || { echo -e "${RED}Not logged in${NC}" >&2; return 1; }

  # Convert named ranges to ISO timestamps
  _range_to_ts() {
    case "$1" in
      today)     echo "$(date -u +%Y-%m-%dT00:00:00Z)|$(date -u +%Y-%m-%dT23:59:59Z)" ;;
      yesterday) echo "$(date -u -d '1 day ago' +%Y-%m-%dT00:00:00Z 2>/dev/null || date -u -v-1d +%Y-%m-%dT00:00:00Z)|$(date -u -d '1 day ago' +%Y-%m-%dT23:59:59Z 2>/dev/null || date -u -v-1d +%Y-%m-%dT23:59:59Z)" ;;
      1h)        echo "$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v-1H +%Y-%m-%dT%H:%M:%SZ)|$(date -u +%Y-%m-%dT%H:%M:%SZ)" ;;
      24h)       echo "$(date -u -d '24 hours ago' +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v-24H +%Y-%m-%dT%H:%M:%SZ)|$(date -u +%Y-%m-%dT%H:%M:%SZ)" ;;
      7d)        echo "$(date -u -d '7 days ago' +%Y-%m-%dT00:00:00Z 2>/dev/null || date -u -v-7d +%Y-%m-%dT00:00:00Z)|$(date -u +%Y-%m-%dT23:59:59Z)" ;;
      *)         echo "$1" ;;  # pass through ISO range like "2026-04-10T00:00:00Z|2026-04-10T23:59:59Z"
    esac
  }

  local ts1 ts2
  ts1=$(_range_to_ts "$range1")
  ts2=$(_range_to_ts "$range2")
  local from1="${ts1%%|*}" to1="${ts1##*|}"
  local from2="${ts2%%|*}" to2="${ts2##*|}"

  _agg_query() {
    local from="$1" to="$2"
    local body="{\"size\":0,\"query\":{\"range\":{\"@timestamp\":{\"gte\":\"$from\",\"lte\":\"$to\"}}},\"aggs\":{\"total\":{\"value_count\":{\"field\":\"_index\"}},\"errors\":{\"filter\":{\"terms\":{\"level.keyword\":[\"ERROR\",\"FATAL\"]}}},\"by_service\":{\"terms\":{\"field\":\"service.keyword\",\"size\":20}},\"by_level\":{\"terms\":{\"field\":\"level.keyword\",\"size\":10}}}}"
    curl -sf -H "Authorization: Bearer ${tok}" -H "Content-Type: application/json" \
      "${PROXY}/${DEFAULT_INDEX}/_search" -d "$body" 2>/dev/null
  }

  echo -e "${BOLD}📊 Diff: ${CYAN}${range1}${NC} vs ${CYAN}${range2}${NC}\n"

  local r1 r2
  r1=$(_agg_query "$from1" "$to1")
  r2=$(_agg_query "$from2" "$to2")

  if [ -z "$r1" ] || [ -z "$r2" ]; then
    echo -e "${RED}Query failed${NC}" >&2; return 1
  fi

  local t1 t2 e1 e2
  t1=$(echo "$r1" | jq '.hits.total.value // 0')
  t2=$(echo "$r2" | jq '.hits.total.value // 0')
  e1=$(echo "$r1" | jq '.aggregations.errors.doc_count // 0')
  e2=$(echo "$r2" | jq '.aggregations.errors.doc_count // 0')

  _delta() {
    local a=$1 b=$2
    local d=$(( a - b ))
    if [ $d -gt 0 ]; then echo -e "${RED}+${d}${NC}"
    elif [ $d -lt 0 ]; then echo -e "${GREEN}${d}${NC}"
    else echo "0"; fi
  }

  printf "  ${BOLD}%-15s %10s %10s %10s${NC}\n" "METRIC" "$range1" "$range2" "DELTA"
  printf "  %-15s %10s %10s %10b\n" "Total docs" "$t1" "$t2" "$(_delta $t1 $t2)"
  printf "  %-15s %10s %10s %10b\n" "Errors" "$e1" "$e2" "$(_delta $e1 $e2)"

  echo -e "\n  ${BOLD}Errors by Service:${NC}"
  printf "  ${BOLD}%-20s %8s %8s %8s${NC}\n" "SERVICE" "$range1" "$range2" "DELTA"
  # Merge service data from both ranges
  local svcs
  svcs=$(echo "$r1" "$r2" | jq -rs '[.[].aggregations.by_service.buckets[].key] | unique | .[]')
  for svc in $svcs; do
    local c1 c2
    c1=$(echo "$r1" | jq -r --arg s "$svc" '[.aggregations.by_service.buckets[] | select(.key==$s) | .doc_count] | .[0] // 0')
    c2=$(echo "$r2" | jq -r --arg s "$svc" '[.aggregations.by_service.buckets[] | select(.key==$s) | .doc_count] | .[0] // 0')
    printf "  %-20s %8s %8s %8b\n" "$svc" "$c1" "$c2" "$(_delta $c1 $c2)"
  done
}

cmd_profile() {
  local tok
  tok=$(get_token) || { echo -e "${RED}Not logged in${NC}" >&2; return 1; }
  # Decode JWT payload
  local payload
  payload=$(echo "$tok" | cut -d. -f2 | tr '_-' '/+' | base64 -d 2>/dev/null) || payload='{}'

  if [ "$IS_TTY" = "false" ]; then echo "$payload" | jq . 2>/dev/null || echo "$payload"; return; fi

  echo -e "${BOLD}🔐 Token Profile${NC}\n"
  local client sub iss exp iat scope
  client=$(echo "$payload" | jq -r '.client_id // .azp // "—"' 2>/dev/null)
  sub=$(echo "$payload" | jq -r '.sub // "—"' 2>/dev/null)
  iss=$(echo "$payload" | jq -r '.iss // "—"' 2>/dev/null)
  exp=$(echo "$payload" | jq -r '.exp // 0' 2>/dev/null)
  iat=$(echo "$payload" | jq -r '.iat // 0' 2>/dev/null)
  scope=$(echo "$payload" | jq -r '.scope // (.scp | join(" ")) // "—"' 2>/dev/null)

  echo -e "  ${BOLD}Client:${NC}  ${CYAN}${client}${NC}"
  echo -e "  ${BOLD}Subject:${NC} ${sub}"
  echo -e "  ${BOLD}Issuer:${NC}  ${iss}"
  echo -e "  ${BOLD}Scopes:${NC}  ${CYAN}${scope}${NC}"

  if [ "$exp" -gt 0 ] 2>/dev/null; then
    local now=$(date +%s) remaining=$(( exp - now ))
    local exp_fmt=$(date -d "@$exp" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -r "$exp" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "$exp")
    echo -e "  ${BOLD}Expires:${NC} ${exp_fmt}"
    if [ $remaining -gt 0 ]; then
      echo -e "  ${BOLD}TTL:${NC}     ${GREEN}${remaining}s remaining${NC}"
    else
      echo -e "  ${BOLD}TTL:${NC}     ${RED}EXPIRED ($(( -remaining ))s ago)${NC}"
    fi
  fi
  if [ "$iat" -gt 0 ] 2>/dev/null; then
    local iat_fmt=$(date -d "@$iat" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -r "$iat" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "$iat")
    echo -e "  ${BOLD}Issued:${NC}  ${iat_fmt}"
  fi

  # Scope breakdown
  if [ "$scope" != "—" ]; then
    echo -e "\n  ${BOLD}Scope Breakdown:${NC}"
    for s in $scope; do
      local icon="🔑"
      echo "$s" | grep -q 'write\|admin' && icon="✏️"
      echo "$s" | grep -q 'read' && icon="👁"
      echo -e "    ${icon}  ${s}"
    done
  fi
}

cmd_install_man() {
  local mandir="${1:-/usr/local/share/man/man1}"
  local src="${PROXY}/docs/oauth4os-demo.1"
  echo -e "${CYAN}Installing man page...${NC}"
  mkdir -p "$mandir" 2>/dev/null || { echo -e "${YELLOW}Need sudo: sudo oauth4os-demo install-man${NC}"; return 1; }
  curl -sf "$src" -o "${mandir}/oauth4os-demo.1" 2>/dev/null || {
    # Fallback: generate inline
    local script_dir="$(cd "$(dirname "$0")" && pwd)"
    if [ -f "${script_dir}/../../docs/oauth4os-demo.1" ]; then
      cp "${script_dir}/../../docs/oauth4os-demo.1" "${mandir}/oauth4os-demo.1"
    else
      echo -e "${RED}Could not download man page${NC}"; return 1
    fi
  }
  echo -e "${GREEN}✓ Installed to ${mandir}/oauth4os-demo.1${NC}"
  echo "  Run: man oauth4os-demo"
}

cmd_top() {
  trap 'tput cnorm; echo; exit 0' INT
  tput civis
  while true; do
    local resp
    resp=$(curl -sf "${PROXY}/admin/analytics" 2>/dev/null)
    local metrics
    metrics=$(curl -sf "${PROXY}/metrics" 2>/dev/null)

    tput clear
    local ts=$(date '+%H:%M:%S')
    echo -e "${BOLD}🔐 oauth4os top${NC}                                              ${CYAN}${ts}${NC}  (q=quit)"
    echo -e "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Metrics summary
    if [ -n "$metrics" ]; then
      local total=$(echo "$metrics" | grep '^oauth4os_requests_total ' | awk '{print $2}')
      local active=$(echo "$metrics" | grep '^oauth4os_requests_active ' | awk '{print $2}')
      local failed=$(echo "$metrics" | grep '^oauth4os_requests_failed ' | awk '{print $2}')
      local shed=$(echo "$metrics" | grep '^oauth4os_loadshed_total ' | awk '{print $2}')
      local cache_h=$(echo "$metrics" | grep '^oauth4os_cache_hits ' | awk '{print $2}')
      local uptime=$(echo "$metrics" | grep '^oauth4os_uptime_seconds ' | awk '{print $2}')
      printf "\n  ${BOLD}Requests:${NC} %-8s  ${BOLD}Active:${NC} ${CYAN}%-4s${NC}  ${BOLD}Failed:${NC} ${RED}%-6s${NC}  ${BOLD}Shed:${NC} %-6s  ${BOLD}Cache:${NC} %-6s  ${BOLD}Up:${NC} %ss\n" \
        "${total:-0}" "${active:-0}" "${failed:-0}" "${shed:-0}" "${cache_h:-0}" "${uptime:-?}"
    fi

    # Top clients
    if [ -n "$resp" ]; then
      echo -e "\n  ${BOLD}Top Clients:${NC}"
      printf "  ${CYAN}%-25s %10s %20s${NC}\n" "CLIENT" "REQUESTS" "LAST SEEN"
      echo "$resp" | jq -r '.top_clients[:10][] | "\(.client_id) \(.requests) \(.last_seen)"' 2>/dev/null | while read -r cid reqs seen; do
        local ago_s=""
        if [ "$seen" != "null" ] && [ -n "$seen" ]; then
          ago_s=$(echo "$seen" | cut -dT -f2 | cut -d. -f1)
        fi
        printf "  %-25s %10s %20s\n" "$cid" "$reqs" "${ago_s:-—}"
      done

      echo -e "\n  ${BOLD}Top Scopes:${NC}"
      echo "$resp" | jq -r '.scope_distribution[:8][] | "\(.name) \(.count)"' 2>/dev/null | while read -r name cnt; do
        printf "  %-30s %8s\n" "$name" "$cnt"
      done
    fi

    read -t 3 -n 1 key 2>/dev/null && [ "$key" = "q" ] && { tput cnorm; echo; break; }
  done
}

cmd_env() {
  echo -e "${BOLD}🔧 Environment${NC}\n"
  echo -e "  ${BOLD}Proxy:${NC}     ${PROXY}"
  echo -e "  ${BOLD}Index:${NC}     ${DEFAULT_INDEX}"
  echo -e "  ${BOLD}Format:${NC}    ${DEFAULT_FORMAT}"
  echo -e "  ${BOLD}Config:${NC}    ${CONFIG_FILE} $([ -f "$CONFIG_FILE" ] && echo "${GREEN}✓${NC}" || echo "${YELLOW}(not created)${NC}")"
  echo -e "  ${BOLD}Token:${NC}     ${TOKEN_FILE} $([ -f "$TOKEN_FILE" ] && echo "${GREEN}✓${NC}" || echo "${YELLOW}(no token)${NC}")"
  echo -e "  ${BOLD}History:${NC}   ${HISTORY_FILE:-~/.oauth4os-history} $([ -f "${HISTORY_FILE:-$HOME/.oauth4os-history}" ] && echo "$(wc -l < "${HISTORY_FILE:-$HOME/.oauth4os-history}") entries" || echo "(empty)")"
  echo -e "  ${BOLD}Bookmarks:${NC} ${BOOKMARKS_FILE:-~/.oauth4os-bookmarks} $([ -f "${BOOKMARKS_FILE:-$HOME/.oauth4os-bookmarks}" ] && echo "$(wc -l < "${BOOKMARKS_FILE:-$HOME/.oauth4os-bookmarks}") saved" || echo "(empty)")"
  echo -e "  ${BOLD}Aliases:${NC}   ${ALIAS_FILE} $([ -f "$ALIAS_FILE" ] && echo "$(wc -l < "$ALIAS_FILE") defined" || echo "(empty)")"
  echo -e "  ${BOLD}TTY:${NC}       $IS_TTY"
  echo -e "  ${BOLD}Deps:${NC}      curl=$(command -v curl >/dev/null && curl --version | head -1 | awk '{print $2}' || echo 'MISSING') jq=$(command -v jq >/dev/null && jq --version 2>/dev/null || echo 'MISSING')"
  echo ""
  echo -ne "  ${BOLD}Proxy:${NC}     "
  if curl -sf --max-time 3 "${PROXY}/health" >/dev/null 2>&1; then
    echo -e "${GREEN}reachable ✓${NC}"
  else
    echo -e "${RED}unreachable ✗${NC}"
  fi
}

cmd_audit() {
  local tok n="${1:-20}"
  tok=$(get_token) || { echo -e "${RED}Not logged in${NC}" >&2; return 1; }
  local resp
  resp=$(curl -sf -H "Authorization: Bearer ${tok}" "${PROXY}/admin/audit?limit=${n}" 2>/dev/null)
  if [ -z "$resp" ]; then
    echo -e "${RED}Failed to fetch audit log${NC}" >&2; return 1
  fi
  if [ "$IS_TTY" = "false" ]; then echo "$resp"; return; fi
  echo -e "${BOLD}📋 Recent Audit Events${NC} (last ${n})\n"
  printf "  ${CYAN}%-20s %-15s %-20s %s${NC}\n" "TIME" "ACTION" "USER" "DETAIL"
  echo "$resp" | jq -r '.events[]? // .[]? | "\(.timestamp // .time) \(.action) \(.user // .actor) \(.detail // .resource // "")"' 2>/dev/null | while read -r ts action user detail; do
    local short_ts=$(echo "$ts" | sed 's/T/ /' | cut -d. -f1)
    printf "  %-20s %-15s %-20s %s\n" "$short_ts" "$action" "$user" "$detail"
  done
}

cmd_alerts() {
  local tok
  tok=$(get_token) || { echo -e "${RED}Not logged in${NC}" >&2; return 1; }
  local resp
  resp=$(curl -sf -H "Authorization: Bearer ${tok}" "${PROXY}/admin/alerts" 2>/dev/null)
  if [ -z "$resp" ]; then
    # Fallback: show metrics-based status
    local metrics
    metrics=$(curl -sf "${PROXY}/metrics" 2>/dev/null)
    if [ -z "$metrics" ]; then echo -e "${RED}Cannot reach proxy${NC}" >&2; return 1; fi
    echo -e "${BOLD}🔔 Alert Status (from metrics)${NC}\n"
    local auth_fail=$(echo "$metrics" | grep '^oauth4os_auth_failed ' | awk '{print $2}')
    local upstream_err=$(echo "$metrics" | grep '^oauth4os_upstream_errors ' | awk '{print $2}')
    local shed=$(echo "$metrics" | grep '^oauth4os_loadshed_total ' | awk '{print $2}')
    local circuit=$(echo "$metrics" | grep '^oauth4os_circuit_opens ' | awk '{print $2}')
    local healthy=$(echo "$metrics" | grep '^oauth4os_upstream_healthy ' | awk '{print $2}')
    local latency=$(echo "$metrics" | grep '^oauth4os_upstream_latency_ms ' | awk '{print $2}')

    _status() { [ "${1:-0}" = "0" ] && echo -e "${GREEN}OK${NC}" || echo -e "${YELLOW}${1}${NC}"; }
    _health() { [ "${1:-1}" = "1" ] && echo -e "${GREEN}healthy${NC}" || echo -e "${RED}DOWN${NC}"; }

    printf "  %-25s %s\n" "Auth failures:" "$(_status $auth_fail)"
    printf "  %-25s %s\n" "Upstream errors:" "$(_status $upstream_err)"
    printf "  %-25s %s\n" "Upstream health:" "$(_health $healthy)"
    printf "  %-25s %s\n" "Upstream latency:" "${latency:-?}ms"
    printf "  %-25s %s\n" "Load shed rejections:" "$(_status $shed)"
    printf "  %-25s %s\n" "Circuit breaker opens:" "$(_status $circuit)"
    return
  fi
  if [ "$IS_TTY" = "false" ]; then echo "$resp"; return; fi
  echo -e "${BOLD}🔔 Active Alerts${NC}\n"
  echo "$resp" | jq -r '.alerts[]? | "\(.state) \(.labels.severity) \(.labels.alertname) \(.annotations.summary)"' 2>/dev/null | while read -r state sev name summary; do
    local icon="🟢"
    [ "$state" = "firing" ] && icon="🔴"
    [ "$state" = "pending" ] && icon="🟡"
    printf "  %s %-10s %-8s %-25s %s\n" "$icon" "$state" "$sev" "$name" "$summary"
  done
}

cmd_latency() {
  local metrics
  metrics=$(curl -sf "${PROXY}/metrics" 2>/dev/null)
  if [ -z "$metrics" ]; then echo -e "${RED}Cannot reach proxy${NC}" >&2; return 1; fi
  local lat=$(echo "$metrics" | grep '^oauth4os_upstream_latency_ms ' | awk '{print $2}')
  local active=$(echo "$metrics" | grep '^oauth4os_requests_active ' | awk '{print $2}')
  local total=$(echo "$metrics" | grep '^oauth4os_requests_total ' | awk '{print $2}')
  local failed=$(echo "$metrics" | grep '^oauth4os_requests_failed ' | awk '{print $2}')
  local uptime=$(echo "$metrics" | grep '^oauth4os_uptime_seconds ' | awk '{print $2}')

  if [ "$IS_TTY" = "false" ]; then
    printf '{"latency_ms":%s,"active":%s,"total":%s,"failed":%s,"uptime_s":%s}\n' \
      "${lat:-0}" "${active:-0}" "${total:-0}" "${failed:-0}" "${uptime:-0}"
    return
  fi

  echo -e "${BOLD}⏱  Latency & Throughput${NC}\n"
  echo -e "  ${BOLD}Upstream latency:${NC}  ${CYAN}${lat:-?}ms${NC}"
  echo -e "  ${BOLD}Active requests:${NC}   ${active:-0}"
  echo -e "  ${BOLD}Total requests:${NC}    ${total:-0}"
  echo -e "  ${BOLD}Failed requests:${NC}   ${RED}${failed:-0}${NC}"
  if [ "${uptime:-0}" -gt 0 ] 2>/dev/null && [ "${total:-0}" -gt 0 ] 2>/dev/null; then
    local rps=$(( total / uptime ))
    local err_pct=0
    [ "$total" -gt 0 ] && err_pct=$(( failed * 100 / total ))
    echo -e "  ${BOLD}Avg throughput:${NC}    ~${rps} req/s"
    echo -e "  ${BOLD}Error rate:${NC}        ${err_pct}%"
  fi
}

cmd_ping() {
  local count="${1:-5}" i=0 total=0 min=999999 max=0
  echo -e "${BOLD}🏓 Pinging ${CYAN}${PROXY}${NC}\n"
  while [ $i -lt $count ]; do
    local start=$(date +%s%N)
    local code=$(curl -sf -o /dev/null -w '%{http_code}' --max-time 5 "${PROXY}/health" 2>/dev/null) || code="000"
    local end=$(date +%s%N)
    local ms=$(( (end - start) / 1000000 ))
    total=$(( total + ms ))
    [ $ms -lt $min ] && min=$ms
    [ $ms -gt $max ] && max=$ms
    if [ "$code" = "200" ]; then
      printf "  ${GREEN}✓${NC} %3dms  HTTP %s\n" "$ms" "$code"
    else
      printf "  ${RED}✗${NC} %3dms  HTTP %s\n" "$ms" "$code"
    fi
    i=$(( i + 1 ))
    [ $i -lt $count ] && sleep 0.5
  done
  local avg=$(( total / count ))
  echo -e "\n  ${BOLD}min/avg/max = ${min}/${avg}/${max} ms${NC}"
}

# Main
ensure_deps
# Strip --json from args (already parsed above)
args=()
for arg in "$@"; do [ "$arg" != "--json" ] && args+=("$arg"); done
set -- "${args[@]}"
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
  stats)    cmd_stats ;;
  export)   shift; cmd_export "$@" ;;
  sql)      shift; cmd_sql "$*" ;;
  history)  cmd_history ;;
  bookmark) shift; cmd_bookmark "$@" ;;
  dashboard|dash) cmd_dashboard ;;
  watch)    shift; cmd_watch "$*" ;;
  diff)     shift; cmd_diff "${1:-today}" "${2:-yesterday}" ;;
  profile)  cmd_profile ;;
  top)      cmd_top ;;
  env)      cmd_env ;;
  audit)    shift; cmd_audit "${1:-20}" ;;
  alerts)   cmd_alerts ;;
  latency)  cmd_latency ;;
  ping)     shift; cmd_ping "${1:-5}" ;;
  install-man) shift; cmd_install_man "${1:-}" ;;
  config)   shift; cmd_config "$@" ;;
  alias)    shift; cmd_alias "$@" ;;
  completion) shift; cmd_completion "${1:-bash}" ;;
  help|-h|--help) usage ;;
  "") usage ;;
  *) echo -e "${RED}Unknown command: $1${NC}"; usage ;;
esac
