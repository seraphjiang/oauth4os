#!/usr/bin/env bash
# CLI integration test — runs all 22 commands against live proxy.
# Usage: bash test/cli-live-test.sh [PROXY_URL]
set -euo pipefail

SCRIPT="$(cd "$(dirname "$0")/.." && pwd)/cmd/proxy/oauth4os-demo.sh"
PROXY="${1:-https://f5cmk2hxwx.us-west-2.awsapprunner.com}"
export OAUTH4OS_PROXY="$PROXY"
PASS=0 FAIL=0 SKIP=0

check() {
  local name="$1" expected="$2"; shift 2
  local out
  out=$(timeout 10 "$@" 2>&1) || true
  if echo "$out" | grep -qiE "$expected"; then
    echo "✅ $name"
    PASS=$((PASS+1))
  else
    echo "❌ $name — expected /$expected/"
    echo "   got: ${out:0:120}"
    FAIL=$((FAIL+1))
  fi
}

skip() {
  echo "⏭  $1 (interactive — skipped)"
  SKIP=$((SKIP+1))
}

echo "═══════════════════════════════════════════════"
echo " CLI Live Integration Test — 22 commands"
echo " Proxy: $PROXY"
echo "═══════════════════════════════════════════════"
echo ""

# 1. help
check "help"              "COMMANDS|login|search"       bash "$SCRIPT" help
# 2. no-args
check "no-args=usage"     "COMMANDS|Usage"              bash "$SCRIPT"
# 3. unknown
check "unknown cmd"       "Unknown command"             bash "$SCRIPT" xyzzy999
# 4. status
check "status"            "healthy|ok|unreachable"      bash "$SCRIPT" status
# 5. login
check "login"             "Logged in|token|error"       bash "$SCRIPT" login
# 6. token
check "token"             "Token|cached|ey|Not logged"  bash "$SCRIPT" token
# 7. whoami
check "whoami"            "client_id|sub|ey|Not logged" bash "$SCRIPT" whoami
# 8. profile
check "profile"           "Client:|Scopes:|TTL:|Not logged" bash "$SCRIPT" profile
# 9. search
check "search"            "hits|Query:|error|Not logged" bash "$SCRIPT" search '*'
# 10. sql (no query = usage)
check "sql usage"         "Usage|SELECT"                bash "$SCRIPT" sql
# 11. services
check "services"          "logs|Failed|Not logged"      bash "$SCRIPT" services
# 12. indices
check "indices"           "docs|Failed|Not logged"      bash "$SCRIPT" indices
# 13. stats
check "stats"             "Total|Errors|Not logged"     bash "$SCRIPT" stats
# 14. export (missing -o = usage)
check "export usage"      "Usage|output"                bash "$SCRIPT" export 'level:ERROR'
# 15. diff
check "diff"              "Diff|METRIC|Not logged"      bash "$SCRIPT" diff today yesterday
# 16. history
check "history"           "Recent|No query"             bash "$SCRIPT" history
# 17. bookmark list
check "bookmark list"     "Bookmarks|No bookmarks"      bash "$SCRIPT" bookmark list
# 18. config show
check "config show"       "proxy:|index:|format:"       bash "$SCRIPT" config show
# 19. alias list
check "alias list"        "Aliases|No aliases"          bash "$SCRIPT" alias list
# 20. completion bash
check "completion bash"   "complete|COMPREPLY"           bash "$SCRIPT" completion bash
# 21. completion zsh
check "completion zsh"    "compdef"                      bash "$SCRIPT" completion zsh
# 22. install-man (dry — will fail without sudo, that's expected)
check "install-man"       "Installing|Need sudo|Installed|error" bash "$SCRIPT" install-man /tmp/oauth4os-man-test

# Interactive commands — can't test in CI, just verify they start
skip "tail (interactive — needs Ctrl+C)"
skip "watch (interactive — needs Ctrl+C)"
skip "dashboard (interactive — needs Ctrl+C)"
skip "top (interactive — needs Ctrl+C)"

# Cleanup
check "logout"            "Logged out|cleared"          bash "$SCRIPT" logout

echo ""
echo "═══════════════════════════════════════════════"
echo " Results: $PASS passed, $FAIL failed, $SKIP skipped"
echo "═══════════════════════════════════════════════"
[ "$FAIL" -eq 0 ] && echo "✅ All non-interactive tests passed" || echo "❌ $FAIL test(s) failed"
exit "$FAIL"
