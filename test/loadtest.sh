#!/usr/bin/env bash
# Load test — 10k requests, measure degradation curve.
# Usage: ./test/loadtest.sh [PROXY_URL]
set -euo pipefail

PROXY="${1:-https://f5cmk2hxwx.us-west-2.awsapprunner.com}"
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

echo "═══════════════════════════════════════"
echo "  oauth4os Load Test (10k requests)"
echo "  Proxy: $PROXY"
echo "═══════════════════════════════════════"

curl -sf "$PROXY/health" >/dev/null 2>&1 || { echo "Proxy not reachable"; exit 1; }

# Run in batches to measure degradation
for BATCH_SIZE in 100 500 1000 2000 3000 3400; do
    CONC=$((BATCH_SIZE / 10))
    [ "$CONC" -lt 5 ] && CONC=5
    [ "$CONC" -gt 100 ] && CONC=100

    # Generate URLs
    for i in $(seq 1 "$BATCH_SIZE"); do
        case $((i % 4)) in
            0) echo "$PROXY/health" ;;
            1) echo "$PROXY/.well-known/openid-configuration" ;;
            2) echo "$PROXY/.well-known/jwks.json" ;;
            3) echo "$PROXY/oauth/register" ;;
        esac
    done > "$TMPDIR/urls.txt"

    START=$(date +%s%N)
    xargs -P "$CONC" -I{} sh -c '
        s=$(date +%s%N); code=$(curl -sf -o /dev/null -w "%{http_code}" "{}" 2>/dev/null || echo "000"); e=$(date +%s%N); echo "$(( (e - s) / 1000000 )) $code"
    ' < "$TMPDIR/urls.txt" > "$TMPDIR/results.txt" 2>/dev/null
    END=$(date +%s%N)
    WALL_MS=$(( (END - START) / 1000000 ))

    TOTAL=$(wc -l < "$TMPDIR/results.txt")
    ERRORS=$(awk '$2 == "000" || $2 >= 500 {c++} END {print c+0}' "$TMPDIR/results.txt")
    P50=$(sort -n "$TMPDIR/results.txt" | awk '{print $1}' | awk -v p=0.50 'NR==1{n=0}{a[n++]=$1}END{print a[int(n*p)]}')
    P95=$(sort -n "$TMPDIR/results.txt" | awk '{print $1}' | awk -v p=0.95 'NR==1{n=0}{a[n++]=$1}END{print a[int(n*p)]}')
    P99=$(sort -n "$TMPDIR/results.txt" | awk '{print $1}' | awk -v p=0.99 'NR==1{n=0}{a[n++]=$1}END{print a[int(n*p)]}')
    RPS=$(awk "BEGIN{printf \"%.0f\", $TOTAL / ($WALL_MS / 1000.0)}")

    printf "  %5d reqs @ %3d conc → %4s req/s | p50: %4sms | p95: %4sms | p99: %4sms | err: %d\n" \
        "$BATCH_SIZE" "$CONC" "$RPS" "$P50" "$P95" "$P99" "$ERRORS"
done

echo "═══════════════════════════════════════"
echo "  Done — check for degradation in p95/p99 as load increases"
echo "═══════════════════════════════════════"
