#!/bin/bash
# Chaos test: serve expired/invalid JWKS
# Starts a fake JWKS endpoint returning bad keys, verifies proxy rejects tokens.
set -euo pipefail

PROXY="${PROXY_URL:-http://localhost:8443}"
JWKS_PORT=9999

echo "=== CHAOS: Expired/invalid JWKS ==="

# 1. Start fake JWKS server returning empty keyset
echo "[1/3] Starting fake JWKS server (empty keys)..."
python3 -c "
import http.server, json, sys
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-Type','application/json')
        self.end_headers()
        self.wfile.write(json.dumps({'keys':[]}).encode())
    def log_message(self,*a): pass
http.server.HTTPServer(('',${JWKS_PORT}),H).serve_forever()
" &
JWKS_PID=$!
sleep 1

echo "  Fake JWKS at http://localhost:$JWKS_PORT"

# 2. Create a JWT signed with a key not in JWKS (any random token)
echo "[2/3] Sending request with token that won't validate..."
FAKE_TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImZha2UifQ.eyJpc3MiOiJodHRwOi8vbG9jYWxob3N0Ojk5OTkiLCJzdWIiOiJ0ZXN0Iiwic2NvcGUiOiJyZWFkOmxvZ3MtKiIsImV4cCI6OTk5OTk5OTk5OX0.fake-signature"

STATUS=$(curl -sf -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $FAKE_TOKEN" \
    "$PROXY/_search" -d '{"query":{"match_all":{}}}' 2>/dev/null || echo "000")

echo "  Response: HTTP $STATUS"
if [ "$STATUS" = "401" ] || [ "$STATUS" = "403" ]; then
    echo "  ✅ Proxy correctly rejected token with invalid JWKS"
else
    echo "  ⚠️ Unexpected status: $STATUS"
fi

# 3. Test JWKS server returning 500
echo "[3/3] Testing JWKS server error..."
kill $JWKS_PID 2>/dev/null || true
sleep 1

python3 -c "
import http.server, sys
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(500)
        self.end_headers()
    def log_message(self,*a): pass
http.server.HTTPServer(('',${JWKS_PORT}),H).serve_forever()
" &
JWKS_PID=$!
sleep 1

STATUS=$(curl -sf -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $FAKE_TOKEN" \
    "$PROXY/_search" -d '{"query":{"match_all":{}}}' 2>/dev/null || echo "000")

echo "  JWKS 500 → Proxy response: HTTP $STATUS"
[ "$STATUS" = "401" ] || [ "$STATUS" = "403" ] || [ "$STATUS" = "500" ] && echo "  ✅ Handled gracefully" || echo "  ⚠️ Unexpected: $STATUS"

kill $JWKS_PID 2>/dev/null || true
echo "Done."
