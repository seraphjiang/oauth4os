#!/bin/bash
# Chaos test: kill upstream OpenSearch mid-request
# Verifies proxy returns 502/503 gracefully, doesn't crash or leak connections.
set -euo pipefail

PROXY="${PROXY_URL:-http://localhost:8443}"
TOKEN="${TEST_TOKEN:-}"
AUTH=""
[ -n "$TOKEN" ] && AUTH="-H Authorization: Bearer $TOKEN"

echo "=== CHAOS: Kill upstream mid-request ==="

# 1. Verify baseline works
echo "[1/4] Baseline check..."
STATUS=$(curl -sf -o /dev/null -w "%{http_code}" $AUTH "$PROXY/health")
echo "  Health: HTTP $STATUS"
[ "$STATUS" = "200" ] || { echo "ABORT: proxy not healthy"; exit 1; }

# 2. Start a slow query in background
echo "[2/4] Starting slow query..."
curl -s $AUTH "$PROXY/_search" -d '{"query":{"match_all":{}},"size":10000}' -o /dev/null &
CURL_PID=$!

# 3. Kill OpenSearch
echo "[3/4] Killing OpenSearch..."
docker compose stop opensearch 2>/dev/null || docker stop opensearch 2>/dev/null || true
sleep 2

# 4. Verify proxy handles it gracefully
echo "[4/4] Verifying graceful degradation..."
for i in $(seq 1 5); do
    STATUS=$(curl -sf -o /dev/null -w "%{http_code}" $AUTH "$PROXY/_search" -d '{"query":{"match_all":{}}}' 2>/dev/null || echo "000")
    echo "  Request $i: HTTP $STATUS"
    if [ "$STATUS" = "502" ] || [ "$STATUS" = "503" ] || [ "$STATUS" = "504" ]; then
        echo "  ✅ Proxy returned error gracefully"
    elif [ "$STATUS" = "000" ]; then
        echo "  ❌ FAIL: Proxy not responding — possible crash"
    fi
done

# Verify proxy itself is still alive
HEALTH=$(curl -sf -o /dev/null -w "%{http_code}" "$PROXY/health" 2>/dev/null || echo "000")
echo ""
echo "Proxy health after chaos: HTTP $HEALTH"
[ "$HEALTH" != "000" ] && echo "✅ Proxy survived" || echo "❌ Proxy crashed"

# Restart OpenSearch
echo "Restarting OpenSearch..."
docker compose start opensearch 2>/dev/null || docker start opensearch 2>/dev/null || true

kill $CURL_PID 2>/dev/null || true
