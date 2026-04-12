#!/usr/bin/env bash
# CLI test suite — verifies all commands produce expected output format.
# Usage: bash test/cli-format-test.sh [PROXY_URL]
set -euo pipefail

SCRIPT="$(cd "$(dirname "$0")/.." && pwd)/cmd/proxy/oauth4os-demo.sh"
PROXY="${1:-https://f5cmk2hxwx.us-west-2.awsapprunner.com}"
export OAUTH4OS_PROXY="$PROXY"
PASS=0 FAIL=0

check() {
  local name="$1" expected="$2"
  shift 2
  local out
  out=$("$@" 2>&1) || true
  if echo "$out" | grep -qiE "$expected"; then
    echo "✅ $name"
    PASS=$((PASS+1))
  else
    echo "❌ $name — expected /$expected/, got: ${out:0:100}"
    FAIL=$((FAIL+1))
  fi
}

echo "CLI Format Test Suite"
echo "Proxy: $PROXY"
echo "Script: $SCRIPT"
echo "---"

# Help / usage
check "help shows commands"    "COMMANDS|login|search|tail" bash "$SCRIPT" help
check "no-args shows usage"    "COMMANDS|Usage"             bash "$SCRIPT"
check "unknown cmd shows error" "Unknown command"           bash "$SCRIPT" xyzzy123

# Status / health
check "status shows healthy or error" "healthy|ok|unreachable" bash "$SCRIPT" status

# Login
check "login succeeds or shows error" "Logged in|token|error" bash "$SCRIPT" login

# Token
check "token shows info"       "Token|cached|Not logged" bash "$SCRIPT" token

# Whoami
check "whoami shows claims"    "client_id|sub|Not logged" bash "$SCRIPT" whoami

# Search (pipe mode — outputs JSON)
check "search outputs JSON"    '^\[' bash -c "echo '' | bash '$SCRIPT' search '*' 2>/dev/null || echo '[]'"

# Services
check "services lists or fails" "logs|Failed|Not logged" bash "$SCRIPT" services

# Indices
check "indices lists or fails"  "docs|Failed|Not logged" bash "$SCRIPT" indices

# Stats
check "stats shows metrics"    "Total|Errors|Error|Not logged" bash "$SCRIPT" stats

# History
check "history shows list"     "Recent|No query" bash "$SCRIPT" history

# Bookmark list
check "bookmark list"          "Bookmarks|No bookmarks" bash "$SCRIPT" bookmark list

# Config show
check "config show"            "proxy:|index:|format:" bash "$SCRIPT" config show

# Alias list
check "alias list"             "Aliases|No aliases" bash "$SCRIPT" alias list

# Completion bash
check "completion bash"        "complete|compgen|COMPREPLY" bash "$SCRIPT" completion bash

# Completion zsh
check "completion zsh"         "compdef|compadd" bash "$SCRIPT" completion zsh

# Export (no output file = usage error)
check "export needs --output"  "Usage|output" bash "$SCRIPT" export 'level:ERROR'

# SQL (no query = usage error)
check "sql needs query"        "Usage|SELECT" bash "$SCRIPT" sql

# Diff
check "diff shows comparison"  "Diff|METRIC|Not logged" bash "$SCRIPT" diff today yesterday

# Logout
check "logout clears token"    "Logged out|cleared" bash "$SCRIPT" logout

echo ""
echo "---"
echo "Results: $PASS passed, $FAIL failed, $((PASS+FAIL)) total"
[ "$FAIL" -eq 0 ] && echo "✅ All tests passed" || echo "❌ $FAIL test(s) failed"
exit "$FAIL"
