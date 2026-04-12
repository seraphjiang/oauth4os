#!/bin/bash
# Chaos test: simulate clock skew
# Verifies proxy correctly rejects tokens that appear expired due to clock drift.
set -euo pipefail

PROXY="${PROXY_URL:-http://localhost:8443}"

echo "=== CHAOS: Clock skew simulation ==="

# Create a token with exp set to 1 second ago (simulates clock skew)
echo "[1/2] Testing token with exp in the past..."
# This is a structurally valid JWT with exp=1 (epoch), will fail exp check
EXPIRED_TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ0ZXN0Iiwic3ViIjoidGVzdCIsImV4cCI6MSwic2NvcGUiOiJyZWFkOmxvZ3MtKiJ9.fake"

STATUS=$(curl -sf -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $EXPIRED_TOKEN" \
    "$PROXY/_search" -d '{"query":{"match_all":{}}}' 2>/dev/null || echo "000")

echo "  Expired token → HTTP $STATUS"
if [ "$STATUS" = "401" ] || [ "$STATUS" = "403" ]; then
    echo "  ✅ Correctly rejected expired token"
else
    echo "  ⚠️ Unexpected: $STATUS (expected 401/403)"
fi

# Test token with no exp claim at all
echo "[2/2] Testing token with no exp claim..."
NO_EXP_TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ0ZXN0Iiwic3ViIjoidGVzdCIsInNjb3BlIjoicmVhZDpsb2dzLSoifQ.fake"

STATUS=$(curl -sf -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $NO_EXP_TOKEN" \
    "$PROXY/_search" -d '{"query":{"match_all":{}}}' 2>/dev/null || echo "000")

echo "  No-exp token → HTTP $STATUS"
if [ "$STATUS" = "401" ] || [ "$STATUS" = "403" ]; then
    echo "  ✅ Correctly rejected token without expiry"
else
    echo "  ⚠️ Unexpected: $STATUS (expected 401/403)"
fi

echo "Done."
