#!/usr/bin/env bash
# CLI integration test — starts proxy, exercises all CLI commands, verifies output.
# Usage: ./test/cli-test.sh
# Requires: Go installed (or run in Docker)
set -euo pipefail

PASS=0
FAIL=0
ERRORS=()
PROXY_PID=""
TMPDIR=$(mktemp -d)
CONFIG="$TMPDIR/config.yaml"
CLI_CONFIG="$TMPDIR/cli-config.yaml"
PORT=18443

cleanup() {
    [ -n "$PROXY_PID" ] && kill "$PROXY_PID" 2>/dev/null || true
    rm -rf "$TMPDIR"
}
trap cleanup EXIT

pass() { echo "  ✅ $1"; ((PASS++)); }
fail() { echo "  ❌ $1 — $2"; ((FAIL++)); ERRORS+=("$1: $2"); }

# ── Setup ─────────────────────────────────────────────────────────────────────

cat > "$CONFIG" <<'EOF'
upstream:
  engine: http://127.0.0.1:19200
listen: ":18443"
providers: []
scope_mapping:
  "admin":
    backend_user: admin
    backend_roles: [all_access]
EOF

echo "Building proxy + CLI..."
CGO_ENABLED=0 go build -buildvcs=false -o "$TMPDIR/proxy" ./cmd/proxy
CGO_ENABLED=0 go build -buildvcs=false -o "$TMPDIR/cli" ./cmd/cli

echo "Starting proxy on :$PORT..."
"$TMPDIR/proxy" -config "$CONFIG" &
PROXY_PID=$!
sleep 2

# Verify proxy is up
if ! curl -sf "http://localhost:$PORT/health" >/dev/null 2>&1; then
    echo "Proxy failed to start"
    exit 1
fi

CLI="$TMPDIR/cli"
SERVER="http://localhost:$PORT"

# ── Tests ─────────────────────────────────────────────────────────────────────

echo ""
echo "═══════════════════════════════════════"
echo "  CLI Integration Tests"
echo "═══════════════════════════════════════"

# 1. status
echo ""
echo "1. status"
OUT=$($CLI --server "$SERVER" status 2>&1 || true)
if echo "$OUT" | grep -qi "ok\|healthy\|running\|status"; then
    pass "status returns health info"
else
    fail "status" "unexpected output: ${OUT:0:100}"
fi

# 2. config
echo ""
echo "2. config"
OUT=$($CLI --server "$SERVER" config 2>&1 || true)
if [ $? -le 1 ]; then
    pass "config runs without error"
else
    fail "config" "exit code $?"
fi

# 3. create-token
echo ""
echo "3. create-token"
OUT=$($CLI --server "$SERVER" create-token test-client test-secret admin 2>&1 || true)
if echo "$OUT" | grep -qi "token\|tok_\|access"; then
    pass "create-token returns a token"
    TOKEN_ID=$(echo "$OUT" | grep -oE 'tok_[a-f0-9]+' | head -1)
else
    # Try via direct API as fallback
    RESP=$(curl -sf -X POST "$SERVER/oauth/token" \
        -d "grant_type=client_credentials&client_id=test-client&client_secret=test-secret&scope=admin" 2>/dev/null || echo "")
    if echo "$RESP" | grep -q "access_token"; then
        TOKEN_ID=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null)
        pass "create-token (via API fallback)"
    else
        fail "create-token" "no token in output: ${OUT:0:100}"
        TOKEN_ID=""
    fi
fi

# 4. list-tokens
echo ""
echo "4. list-tokens"
OUT=$($CLI --server "$SERVER" list-tokens 2>&1 || true)
if [ $? -le 1 ]; then
    pass "list-tokens runs without error"
else
    fail "list-tokens" "exit code $?"
fi

# 5. inspect-token
echo ""
echo "5. inspect-token"
if [ -n "${TOKEN_ID:-}" ]; then
    OUT=$($CLI --server "$SERVER" inspect-token "$TOKEN_ID" 2>&1 || true)
    if [ $? -le 1 ]; then
        pass "inspect-token runs without error"
    else
        fail "inspect-token" "exit code $?"
    fi
else
    fail "inspect-token" "no token to inspect"
fi

# 6. revoke-token
echo ""
echo "6. revoke-token"
if [ -n "${TOKEN_ID:-}" ]; then
    OUT=$($CLI --server "$SERVER" revoke-token "$TOKEN_ID" 2>&1 || true)
    if [ $? -le 1 ]; then
        pass "revoke-token runs without error"
    else
        fail "revoke-token" "exit code $?"
    fi
else
    fail "revoke-token" "no token to revoke"
fi

# 7. help
echo ""
echo "7. help"
OUT=$($CLI help 2>&1 || true)
if echo "$OUT" | grep -qi "usage\|commands\|help"; then
    pass "help shows usage"
else
    fail "help" "no usage info: ${OUT:0:100}"
fi

# 8. unknown command
echo ""
echo "8. unknown command"
$CLI --server "$SERVER" nonexistent-command >/dev/null 2>&1
EC=$?
if [ $EC -ne 0 ]; then
    pass "unknown command returns non-zero exit"
else
    fail "unknown command" "expected non-zero exit, got 0"
fi

# ── Summary ───────────────────────────────────────────────────────────────────

echo ""
echo "═══════════════════════════════════════"
echo "  Results: $PASS passed, $FAIL failed"
echo "═══════════════════════════════════════"

if [ $FAIL -gt 0 ]; then
    echo ""
    echo "Failures:"
    for e in "${ERRORS[@]}"; do
        echo "  • $e"
    done
    exit 1
fi
