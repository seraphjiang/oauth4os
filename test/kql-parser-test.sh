#!/usr/bin/env bash
# Unit tests for the KQL-to-DSL parser in oauth4os-demo.
set -euo pipefail

SCRIPT="$(cd "$(dirname "$0")/.." && pwd)/cmd/proxy/oauth4os-demo.sh"
PASS=0 FAIL=0

# Source just the kql_to_dsl function
eval "$(sed -n '/^kql_to_dsl()/,/^}/p' "$SCRIPT")"

check() {
  local name="$1" input="$2" expected="$3"
  local got
  got=$(kql_to_dsl "$input")
  if echo "$got" | grep -qF "$expected"; then
    PASS=$((PASS+1))
  else
    echo "❌ $name"
    echo "   input:    $input"
    echo "   expected: $expected"
    echo "   got:      $got"
    FAIL=$((FAIL+1))
  fi
}

echo "KQL Parser Unit Tests"
echo "---"

check "wildcard star"       "*"                          '{"match_all":{}}'
check "empty string"        ""                           '{"match_all":{}}'
check "simple match"        "level:ERROR"                '"match":{"level":"ERROR"}'
check "range gt"            "latency_ms:>500"            '"range":{"latency_ms":{"gt":500}}'
check "range lt"            "latency_ms:<100"            '"range":{"latency_ms":{"lt":100}}'
check "range gte"           "latency_ms:>=200"           '"range":{"latency_ms":{"gte":200}}'
check "range lte"           "latency_ms:<=50"            '"range":{"latency_ms":{"lte":50}}'
check "wildcard value"      "service:pay*"               '"wildcard"'
check "AND two terms"       "service:payment AND level:ERROR" '"bool"'
check "AND has must"        "service:payment AND level:ERROR" '"must"'
check "NOT term"            "service:auth AND NOT level:INFO" '"must_not"'
check "OR term"             "level:ERROR OR level:WARN"  '"should"'
check "free text"           "timeout"                    '"multi_match"'
check "quoted value"        'message:"connection refused"' '"match"'
check "multiple words"      "connection timeout"         '"multi_match"'
check "wildcard prefix"     "service:*-prod"             '"wildcard"'
check "range with AND"      "latency_ms:>100 AND level:ERROR" '"must"'
check "triple AND"          "a:1 AND b:2 AND c:3"       '"must"'
check "NOT only"            "NOT level:DEBUG"            '"must_not"'

echo ""
echo "---"
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] && echo "✅ All KQL parser tests passed" || echo "❌ $FAIL test(s) failed"
exit "$FAIL"
