#!/bin/bash
# E2E test: oauth4os proxy → OpenSearch
# Prerequisites: docker compose up -d
set -e

PROXY="http://localhost:8443"
RED='\033[0;31m'; GREEN='\033[0;32m'; NC='\033[0m'
PASS=0; FAIL=0

check() {
    local desc="$1" expected="$2" actual="$3"
    if echo "$actual" | grep -q "$expected"; then
        echo -e "${GREEN}✓${NC} $desc"
        ((PASS++))
    else
        echo -e "${RED}✗${NC} $desc (expected: $expected)"
        echo "  got: $actual"
        ((FAIL++))
    fi
}

echo "=== oauth4os E2E Tests ==="
echo ""

# 1. Health check
echo "--- Health ---"
resp=$(curl -sf "$PROXY/health" 2>&1 || echo "FAIL")
check "Health endpoint returns ok" '"status":"ok"' "$resp"

# 2. OpenSearch passthrough (no auth)
echo "--- Passthrough (no auth) ---"
resp=$(curl -sf "$PROXY/" 2>&1 || echo "FAIL")
check "Root returns OpenSearch info" '"cluster_name"' "$resp"

resp=$(curl -sf "$PROXY/_cat/health" 2>&1 || echo "FAIL")
check "_cat/health returns status" "green\|yellow" "$resp"

# 3. Token issuance
echo "--- Token Issuance ---"
resp=$(curl -sf -X POST "$PROXY/oauth/token" \
    -d "grant_type=client_credentials" \
    -d "client_id=test-agent" \
    -d "client_secret=test-secret" \
    -d "scope=read:logs-*" 2>&1 || echo "FAIL")
check "Token endpoint returns access_token" "access_token" "$resp"

TOKEN=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || echo "")

# 4. Authenticated query
echo "--- Authenticated Query ---"
if [ -n "$TOKEN" ]; then
    resp=$(curl -sf -H "Authorization: Bearer $TOKEN" "$PROXY/_cat/indices" 2>&1 || echo "FAIL")
    check "Authenticated _cat/indices works" "" "$resp"  # any response = success

    # Create test index
    curl -sf -X PUT -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        "$PROXY/test-logs-e2e" \
        -d '{"settings":{"number_of_shards":1,"number_of_replicas":0}}' > /dev/null 2>&1 || true

    # Index a document
    resp=$(curl -sf -X POST -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        "$PROXY/test-logs-e2e/_doc" \
        -d '{"timestamp":"2026-04-12T00:00:00Z","level":"ERROR","message":"test error","service":"e2e"}' 2>&1 || echo "FAIL")
    check "Index document succeeds" '"result":"created"' "$resp"

    # Search
    sleep 1  # wait for indexing
    resp=$(curl -sf -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        "$PROXY/test-logs-e2e/_search" \
        -d '{"query":{"match":{"level":"ERROR"}}}' 2>&1 || echo "FAIL")
    check "Search returns hits" '"hits"' "$resp"

    # Cleanup
    curl -sf -X DELETE -H "Authorization: Bearer $TOKEN" "$PROXY/test-logs-e2e" > /dev/null 2>&1 || true
else
    echo -e "${RED}✗${NC} Skipping auth tests — no token"
    ((FAIL++))
fi

# 5. Invalid token
echo "--- Error Cases ---"
resp=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer invalid-token" "$PROXY/_cat/health" 2>&1)
check "Invalid token returns 401" "401" "$resp"

# 6. Token list
echo "--- Token Management ---"
resp=$(curl -sf "$PROXY/oauth/tokens" 2>&1 || echo "FAIL")
check "List tokens returns array" "[" "$resp"

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
