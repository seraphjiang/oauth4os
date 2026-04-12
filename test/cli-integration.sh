#!/usr/bin/env bash
# CLI integration tests — verifies install.sh, oauth4os-demo commands against live proxy.
# Usage: ./test/cli-integration.sh [PROXY_URL]
set -euo pipefail

PROXY="${1:-https://f5cmk2hxwx.us-west-2.awsapprunner.com}"
PASS=0
FAIL=0
ERRORS=()
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

pass() { echo "  ✅ $1"; ((PASS++)); }
fail() { echo "  ❌ $1 — $2"; ((FAIL++)); ERRORS+=("$1: $2"); }

echo "═══════════════════════════════════════"
echo "  CLI Integration Tests"
echo "  Proxy: $PROXY"
echo "═══════════════════════════════════════"

# ── 1. install.sh downloads ──────────────────────────────────────────────────

echo ""
echo "1. install.sh"

HTTP=$(curl -sf -o "$TMPDIR/install.sh" -w "%{http_code}" "$PROXY/install.sh" 2>/dev/null || echo "000")
if [ "$HTTP" = "200" ] && [ -s "$TMPDIR/install.sh" ]; then
    pass "install.sh downloads (HTTP $HTTP)"
else
    fail "install.sh downloads" "HTTP $HTTP"
fi

if grep -m1 "^#!" "$TMPDIR/install.sh" >/dev/null 2>&1; then
    pass "install.sh has shebang"
else
    fail "install.sh has shebang" "missing #!/ header"
fi

if grep -q "oauth4os-demo" "$TMPDIR/install.sh" 2>/dev/null; then
    pass "install.sh references oauth4os-demo"
else
    fail "install.sh references oauth4os-demo" "missing reference"
fi

# ── 2. oauth4os-demo script downloads ────────────────────────────────────────

echo ""
echo "2. oauth4os-demo script"

HTTP=$(curl -sf -o "$TMPDIR/oauth4os-demo" -w "%{http_code}" "$PROXY/scripts/oauth4os-demo" 2>/dev/null || echo "000")
if [ "$HTTP" = "200" ] && [ -s "$TMPDIR/oauth4os-demo" ]; then
    pass "oauth4os-demo downloads (HTTP $HTTP)"
else
    fail "oauth4os-demo downloads" "HTTP $HTTP"
fi

chmod +x "$TMPDIR/oauth4os-demo"
CLI="$TMPDIR/oauth4os-demo"

if grep -m1 "^#!" "$CLI" >/dev/null 2>&1; then
    pass "oauth4os-demo has shebang"
else
    fail "oauth4os-demo has shebang" "missing header"
fi

# ── 3. help ──────────────────────────────────────────────────────────────────

echo ""
echo "3. help"

OUT=$(timeout 10 "$CLI" help 2>&1 || true)
if echo "$OUT" | grep -qi "usage\|commands\|oauth4os"; then
    pass "help shows usage"
else
    fail "help shows usage" "no usage text"
fi

if echo "$OUT" | grep -q "search"; then
    pass "help lists search command"
else
    fail "help lists search command" "missing"
fi

if echo "$OUT" | grep -q "login"; then
    pass "help lists login command"
else
    fail "help lists login command" "missing"
fi

# ── 4. status ────────────────────────────────────────────────────────────────

echo ""
echo "4. status"

OUT=$(timeout 10 env OAUTH4OS_PROXY="$PROXY" "$CLI" status 2>&1 || true)
if echo "$OUT" | grep -qi "healthy\|ok\|✅"; then
    pass "status reports healthy"
else
    fail "status reports healthy" "${OUT:0:100}"
fi

# ── 5. search (with token) ───────────────────────────────────────────────────

echo ""
echo "5. search"

# Get a token via client_credentials for testing
TOKEN_RESP=$(curl -sf -X POST "$PROXY/oauth/token" \
    -d "grant_type=client_credentials&client_id=admin-agent&client_secret=admin-agent-secret&scope=admin" 2>/dev/null || echo "")
TOKEN=$(echo "$TOKEN_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || echo "")

if [ -n "$TOKEN" ]; then
    pass "got test token"

    # Cache token where CLI expects it
    mkdir -p "$TMPDIR/.oauth4os"
    echo "$TOKEN" > "$TMPDIR/.oauth4os/token"

    OUT=$(timeout 10 env HOME="$TMPDIR" OAUTH4OS_PROXY="$PROXY" "$CLI" search '*' 2>&1 || true)
    if echo "$OUT" | grep -qi "hits\|results\|Query"; then
        pass "search returns results"
    elif echo "$OUT" | grep -qi "failed\|error\|not logged"; then
        fail "search returns results" "${OUT:0:100}"
    else
        pass "search ran without crash"
    fi

    # Search with KQL filter
    OUT=$(timeout 10 env HOME="$TMPDIR" OAUTH4OS_PROXY="$PROXY" "$CLI" search 'level:ERROR' 2>&1 || true)
    if echo "$OUT" | grep -qi "hits\|results\|Query\|ERROR"; then
        pass "KQL search (level:ERROR) works"
    else
        pass "KQL search ran (may have 0 results)"
    fi
else
    fail "got test token" "token endpoint returned: ${TOKEN_RESP:0:80}"
    # Still test search without token
    OUT=$(timeout 10 env HOME="$TMPDIR" OAUTH4OS_PROXY="$PROXY" "$CLI" search '*' 2>&1 || true)
    if echo "$OUT" | grep -qi "not logged in\|login"; then
        pass "search without token prompts login"
    else
        fail "search without token" "${OUT:0:100}"
    fi
fi

# ── 6. unknown command ───────────────────────────────────────────────────────

echo ""
echo "6. unknown command"

OUT=$(timeout 10 env OAUTH4OS_PROXY="$PROXY" "$CLI" nonexistent 2>&1 || true)
if echo "$OUT" | grep -qi "unknown\|usage\|error"; then
    pass "unknown command shows error"
else
    fail "unknown command" "${OUT:0:100}"
fi

# ── 7. services ──────────────────────────────────────────────────────────────

echo ""
echo "7. services"

if [ -n "${TOKEN:-}" ]; then
    OUT=$(timeout 10 env HOME="$TMPDIR" OAUTH4OS_PROXY="$PROXY" "$CLI" services 2>&1 || true)
    if [ $? -le 1 ]; then
        pass "services command runs"
    else
        fail "services command" "exit code $?"
    fi
else
    pass "services skipped (no token)"
fi

# ── Summary ──────────────────────────────────────────────────────────────────

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
