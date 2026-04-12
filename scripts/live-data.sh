#!/usr/bin/env bash
# live-data.sh — Simulate live log ingestion: 10 entries every 60s.
# Usage: PROXY_URL=https://... CLIENT_ID=... CLIENT_SECRET=... ./scripts/live-data.sh
set -euo pipefail

PROXY="${PROXY_URL:-https://f5cmk2hxwx.us-west-2.awsapprunner.com}"
CID="${CLIENT_ID:?CLIENT_ID required}"
CSEC="${CLIENT_SECRET:?CLIENT_SECRET required}"
INDEX="logs-demo"
INTERVAL="${INTERVAL:-60}"

get_token() {
  curl -sf -X POST "$PROXY/oauth/token" \
    -d "grant_type=client_credentials&client_id=$CID&client_secret=$CSEC&scope=read:logs-* write:logs-*" \
    | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4
}

SERVICES=("payment" "auth" "cart" "shipping" "inventory")
LEVELS=("INFO" "INFO" "INFO" "WARN" "ERROR")
MSGS_INFO=("Request processed" "Cache hit" "Health check OK" "Connection established" "Task completed")
MSGS_WARN=("Slow response" "Retry attempt" "High memory usage" "Rate limit approaching" "Stale cache")
MSGS_ERROR=("Connection timeout" "Internal server error" "Database unreachable" "Out of memory" "Unhandled exception")

TOKEN=$(get_token)
echo "Live data simulator started — $INTERVAL s interval"

BATCH=0
while true; do
  BATCH=$((BATCH + 1))
  BULK=""
  NOW=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  for i in $(seq 1 10); do
    svc="${SERVICES[$((RANDOM % 5))]}"
    roll=$((RANDOM % 10))
    if [ $roll -lt 6 ]; then lvl="INFO"; msgs=("${MSGS_INFO[@]}")
    elif [ $roll -lt 8 ]; then lvl="WARN"; msgs=("${MSGS_WARN[@]}")
    else lvl="ERROR"; msgs=("${MSGS_ERROR[@]}"); fi
    msg="${msgs[$((RANDOM % 5))]}"
    sc=200; [ "$lvl" = "WARN" ] && sc=$((400 + RANDOM % 30)); [ "$lvl" = "ERROR" ] && sc=$((500 + RANDOM % 4))
    dur=$((RANDOM % 500))
    BULK+='{"index":{"_index":"'"$INDEX"'"}}'$'\n'
    BULK+='{"@timestamp":"'"$NOW"'","service":"'"$svc"'","level":"'"$lvl"'","message":"'"$msg"'","status_code":'"$sc"',"duration_ms":'"$dur"',"trace_id":"trace-'"$(printf '%08x' $RANDOM$RANDOM)"'","user_id":"user-'"$(printf '%03d' $((RANDOM % 50)))"'"}'$'\n'
  done

  HTTP=$(curl -sf -o /dev/null -w "%{http_code}" -X POST "$PROXY/_bulk" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/x-ndjson" \
    -d "$BULK" 2>/dev/null || echo "000")

  if [ "$HTTP" = "401" ]; then
    TOKEN=$(get_token)
    curl -sf -o /dev/null -X POST "$PROXY/_bulk" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/x-ndjson" \
      -d "$BULK"
  fi

  echo "[$(date -u +%H:%M:%S)] Batch $BATCH: 10 docs indexed (HTTP $HTTP)"
  sleep "$INTERVAL"
done
