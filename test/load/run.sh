#!/bin/bash
# Load test for oauth4os proxy using vegeta (https://github.com/tsenart/vegeta)
#
# Install: go install github.com/tsenart/vegeta@latest
# Usage:  ./test/load/run.sh [rate] [duration]
# Example: ./test/load/run.sh 1000 30s
#
# Requires: proxy running at localhost:8443

set -e

PROXY="${PROXY_URL:-http://localhost:8443}"
RATE="${1:-100}"
DURATION="${2:-10s}"
RESULTS_DIR="test/load/results"
mkdir -p "$RESULTS_DIR"

echo "=== oauth4os load test ==="
echo "Target:   $PROXY"
echo "Rate:     $RATE req/s"
echo "Duration: $DURATION"
echo ""

# Check proxy is up
if ! curl -sf "$PROXY/health" > /dev/null 2>&1; then
    echo "ERROR: Proxy not reachable at $PROXY"
    echo "Start with: docker compose up -d"
    exit 1
fi

# Get a token for authenticated tests
TOKEN=$(curl -sf -X POST "$PROXY/oauth/token" \
    -d "grant_type=client_credentials" \
    -d "client_id=loadtest" \
    -d "client_secret=secret" \
    -d "scope=read:logs-*" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "WARNING: Could not get token, running unauthenticated tests only"
fi

# --- Test 1: Health endpoint (baseline) ---
echo "--- Test 1: Health endpoint (baseline) ---"
echo "GET $PROXY/health" | \
    vegeta attack -rate="$RATE" -duration="$DURATION" | \
    vegeta report -type=text | tee "$RESULTS_DIR/health.txt"
echo ""

# --- Test 2: Passthrough (no auth) ---
echo "--- Test 2: Passthrough (no auth) ---"
echo "GET $PROXY/" | \
    vegeta attack -rate="$RATE" -duration="$DURATION" | \
    vegeta report -type=text | tee "$RESULTS_DIR/passthrough.txt"
echo ""

# --- Test 3: Bearer auth proxy ---
if [ -n "$TOKEN" ]; then
    echo "--- Test 3: Bearer auth (token → scope → Cedar → upstream) ---"
    printf "GET %s/_cat/indices\nAuthorization: Bearer %s\n" "$PROXY" "$TOKEN" | \
        vegeta attack -rate="$RATE" -duration="$DURATION" | \
        vegeta report -type=text | tee "$RESULTS_DIR/bearer.txt"
    echo ""
fi

# --- Test 4: Token issuance ---
echo "--- Test 4: Token issuance ---"
printf "POST %s/oauth/token\nContent-Type: application/x-www-form-urlencoded\n@%s\n" \
    "$PROXY" <(echo "grant_type=client_credentials&client_id=loadtest&client_secret=secret&scope=read:logs-*") | \
    vegeta attack -rate="$(( RATE / 10 ))" -duration="$DURATION" | \
    vegeta report -type=text | tee "$RESULTS_DIR/token-issue.txt"
echo ""

# --- Test 5: Invalid token (401 path) ---
echo "--- Test 5: Invalid token (401 path) ---"
printf "GET %s/_search\nAuthorization: Bearer invalid.jwt.token\n" "$PROXY" | \
    vegeta attack -rate="$RATE" -duration="$DURATION" | \
    vegeta report -type=text | tee "$RESULTS_DIR/invalid-token.txt"
echo ""

# --- Summary ---
echo "=== Results saved to $RESULTS_DIR/ ==="
echo ""
echo "For latency histograms:"
echo "  cat $RESULTS_DIR/health.txt"
echo ""
echo "For higher rates:"
echo "  $0 1000 30s   # 1K rps for 30s"
echo "  $0 10000 60s  # 10K rps for 60s"
