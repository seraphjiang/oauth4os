#!/usr/bin/env bash
# Stress test — 1000 concurrent requests through the proxy.
# Usage: ./test/stress.sh [PROXY_URL] [TOTAL] [CONCURRENCY]
set -euo pipefail

PROXY="${1:-http://localhost:8443}"
TOTAL="${2:-1000}"
CONC="${3:-50}"

echo "═══════════════════════════════════════"
echo "  oauth4os Stress Test"
echo "  Proxy: $PROXY"
echo "  Requests: $TOTAL, Concurrency: $CONC"
echo "═══════════════════════════════════════"

# Verify proxy is up
curl -sf "$PROXY/health" >/dev/null 2>&1 || { echo "Proxy not reachable"; exit 1; }

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

ENDPOINTS=("/health" "/.well-known/openid-configuration" "/.well-known/jwks.json" "/oauth/register")

# Generate URL list
for i in $(seq 1 "$TOTAL"); do
    idx=$((i % ${#ENDPOINTS[@]}))
    echo "$PROXY${ENDPOINTS[$idx]}"
done > "$TMPDIR/urls.txt"

START=$(date +%s%N)

# Run with xargs for concurrency, capture timing
xargs -P "$CONC" -I{} sh -c '
    s=$(date +%s%N)
    code=$(curl -sf -o /dev/null -w "%{http_code}" "{}" 2>/dev/null || echo "000")
    e=$(date +%s%N)
    ms=$(( (e - s) / 1000000 ))
    echo "$ms $code"
' < "$TMPDIR/urls.txt" > "$TMPDIR/results.txt" 2>/dev/null

END=$(date +%s%N)
WALL_MS=$(( (END - START) / 1000000 ))

# Parse results
TOTAL_DONE=$(wc -l < "$TMPDIR/results.txt")
ERRORS=$(awk '$2 == "000" || $2 >= 500 {c++} END {print c+0}' "$TMPDIR/results.txt")
SUCCESS=$((TOTAL_DONE - ERRORS))

# Latency percentiles
sort -n "$TMPDIR/results.txt" | awk '{print $1}' > "$TMPDIR/latencies.txt"
P50=$(awk -v p=0.50 'NR==1{n=0} {a[n++]=$1} END{print a[int(n*p)]}' "$TMPDIR/latencies.txt")
P95=$(awk -v p=0.95 'NR==1{n=0} {a[n++]=$1} END{print a[int(n*p)]}' "$TMPDIR/latencies.txt")
P99=$(awk -v p=0.99 'NR==1{n=0} {a[n++]=$1} END{print a[int(n*p)]}' "$TMPDIR/latencies.txt")

THROUGHPUT=$(awk "BEGIN{printf \"%.0f\", $TOTAL_DONE / ($WALL_MS / 1000.0)}")

echo ""
echo "═══════════════════════════════════════"
echo "  Results"
echo "═══════════════════════════════════════"
echo "  Requests:    $TOTAL_DONE completed"
echo "  Successful:  $SUCCESS"
echo "  Errors:      $ERRORS"
echo "  Wall time:   ${WALL_MS}ms"
echo "  Throughput:  ${THROUGHPUT} req/s"
echo "  Latency p50: ${P50}ms"
echo "  Latency p95: ${P95}ms"
echo "  Latency p99: ${P99}ms"
echo "═══════════════════════════════════════"

# Exit non-zero if error rate > 5%
ERROR_PCT=$(awk "BEGIN{printf \"%.1f\", $ERRORS / $TOTAL_DONE * 100}")
if awk "BEGIN{exit !($ERRORS / $TOTAL_DONE > 0.05)}"; then
    echo "❌ Error rate ${ERROR_PCT}% exceeds 5% threshold"
    exit 1
fi
echo "✅ Error rate ${ERROR_PCT}%"
