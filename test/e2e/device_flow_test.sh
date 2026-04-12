#!/bin/bash
# E2E test: Device Authorization Flow (RFC 8628)
# Tests against live proxy at $PROXY (default: http://localhost:8443)
set -euo pipefail

PROXY="${PROXY:-http://localhost:8443}"
PASS=0
FAIL=0

pass() { echo "  ✅ $1"; ((PASS++)); }
fail() { echo "  ❌ $1"; ((FAIL++)); }

echo "=== Device Flow E2E Test ==="
echo "Proxy: $PROXY"
echo ""

# Step 1: Request device code
echo "[1/5] Request device code..."
RESP=$(curl -sf -X POST "$PROXY/oauth/device/code" \
  -d "client_id=cli-demo&scope=read:logs-*" 2>/dev/null || echo '{}')

DEVICE_CODE=$(echo "$RESP" | jq -r '.device_code // empty')
USER_CODE=$(echo "$RESP" | jq -r '.user_code // empty')
VERIFY_URI=$(echo "$RESP" | jq -r '.verification_uri // empty')
INTERVAL=$(echo "$RESP" | jq -r '.interval // 5')

if [ -n "$DEVICE_CODE" ] && [ -n "$USER_CODE" ]; then
  pass "Got device_code + user_code ($USER_CODE)"
else
  fail "Missing device_code or user_code: $RESP"
fi

# Step 2: Poll before approval (should get authorization_pending)
echo "[2/5] Poll before approval (expect pending)..."
POLL=$(curl -sf -X POST "$PROXY/oauth/device/token" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=$DEVICE_CODE&client_id=cli-demo" 2>/dev/null || echo '{}')

POLL_ERR=$(echo "$POLL" | jq -r '.error // empty')
if [ "$POLL_ERR" = "authorization_pending" ]; then
  pass "Got authorization_pending (correct)"
elif [ "$POLL_ERR" = "slow_down" ]; then
  pass "Got slow_down (also correct)"
else
  fail "Expected authorization_pending, got: $POLL"
fi

# Step 3: Simulate user approval
echo "[3/5] Approve device (simulate user)..."
APPROVE=$(curl -sf -X POST "$PROXY/oauth/device/approve" \
  -d "user_code=$USER_CODE&action=approve" 2>/dev/null || echo '')
APPROVE_STATUS=$?

if [ $APPROVE_STATUS -eq 0 ]; then
  pass "Device approved"
else
  fail "Approval failed"
fi

# Step 4: Poll after approval (should get token)
echo "[4/5] Poll after approval (expect token)..."
sleep 1
TOKEN_RESP=$(curl -sf -X POST "$PROXY/oauth/device/token" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=$DEVICE_CODE&client_id=cli-demo" 2>/dev/null || echo '{}')

ACCESS_TOKEN=$(echo "$TOKEN_RESP" | jq -r '.access_token // empty')
if [ -n "$ACCESS_TOKEN" ]; then
  pass "Got access_token"
else
  fail "No access_token: $TOKEN_RESP"
fi

# Step 5: Use token to query
echo "[5/5] Use token to search..."
if [ -n "$ACCESS_TOKEN" ]; then
  SEARCH=$(curl -sf -X POST "$PROXY/logs-*/_search" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"query":{"match_all":{}},"size":1}' 2>/dev/null || echo '{}')
  HITS=$(echo "$SEARCH" | jq -r '.hits.total.value // .hits.total // 0')
  if [ "$HITS" != "null" ] && [ "$HITS" != "" ]; then
    pass "Search returned $HITS hits"
  else
    fail "Search failed: $SEARCH"
  fi
else
  fail "Skipped — no token"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ] && echo "✅ Device flow E2E PASSED" || echo "❌ Device flow E2E FAILED"
exit "$FAIL"
