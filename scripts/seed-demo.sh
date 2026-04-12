#!/usr/bin/env bash
# seed-demo.sh — Index 500 sample log entries via oauth4os proxy.
# Usage: PROXY_URL=http://localhost:8443 ./scripts/seed-demo.sh
set -euo pipefail

PROXY="${PROXY_URL:-http://localhost:8443}"
CLIENT_ID="${CLIENT_ID:-demo}"
CLIENT_SECRET="${CLIENT_SECRET:-secret}"
INDEX="logs-demo"

echo "=== oauth4os demo seeder ==="
echo "Proxy: $PROXY"

# Get token
echo -n "Getting token... "
TOKEN=$(curl -sf -X POST "$PROXY/oauth/token" \
  -d "grant_type=client_credentials&client_id=$CLIENT_ID&client_secret=$CLIENT_SECRET&scope=read:logs-* write:logs-*" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])" 2>/dev/null \
  || curl -sf -X POST "$PROXY/oauth/token" \
  -d "grant_type=client_credentials&client_id=$CLIENT_ID&client_secret=$CLIENT_SECRET&scope=read:logs-* write:logs-*" \
  | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
echo "OK ($TOKEN)"

# Create index
echo -n "Creating index... "
curl -sf -X PUT "$PROXY/$INDEX" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "settings": {"number_of_shards": 1, "number_of_replicas": 0},
    "mappings": {"properties": {
      "@timestamp": {"type": "date"},
      "service": {"type": "keyword"},
      "level": {"type": "keyword"},
      "message": {"type": "text"},
      "status_code": {"type": "integer"},
      "duration_ms": {"type": "float"},
      "trace_id": {"type": "keyword"},
      "user_id": {"type": "keyword"}
    }}
  }' > /dev/null 2>&1 || true
echo "OK"

# Services and their log patterns
SERVICES=("payment" "auth" "cart" "shipping" "inventory")
LEVELS=("INFO" "INFO" "INFO" "WARN" "ERROR")  # weighted: 60% info, 20% warn, 20% error

# Message templates per service
payment_info=("Payment processed successfully" "Refund initiated" "Invoice generated" "Payment method validated" "Subscription renewed")
payment_warn=("Payment retry attempt" "Slow payment gateway response" "Currency conversion fallback")
payment_error=("Payment gateway timeout" "Insufficient funds" "Card declined" "Fraud detection triggered")

auth_info=("User login successful" "Token refreshed" "MFA verified" "Session created" "Password reset email sent")
auth_warn=("Failed login attempt" "Token near expiry" "Unusual login location detected")
auth_error=("Authentication failed" "Token validation error" "MFA timeout" "Account locked")

cart_info=("Item added to cart" "Cart updated" "Coupon applied" "Cart checkout started" "Wishlist item moved to cart")
cart_warn=("Item out of stock" "Price changed since added" "Cart size limit approaching")
cart_error=("Cart sync failed" "Inventory check timeout" "Price calculation error")

shipping_info=("Shipment created" "Tracking updated" "Delivery confirmed" "Label generated" "Carrier assigned")
shipping_warn=("Delivery delayed" "Address validation warning" "Carrier rate limit")
shipping_error=("Shipping API error" "Address not found" "Label generation failed")

inventory_info=("Stock updated" "Reorder triggered" "Warehouse sync complete" "SKU registered" "Batch received")
inventory_warn=("Low stock alert" "Sync delay detected" "Warehouse capacity warning")
inventory_error=("Stock count mismatch" "Warehouse API timeout" "Duplicate SKU detected")

# Generate bulk payload
echo -n "Generating 500 log entries... "
BULK=""
NOW=$(date +%s)
for i in $(seq 1 500); do
  # Pick service
  svc_idx=$((RANDOM % 5))
  svc="${SERVICES[$svc_idx]}"

  # Pick level (weighted)
  roll=$((RANDOM % 10))
  if [ $roll -lt 6 ]; then level="INFO"
  elif [ $roll -lt 8 ]; then level="WARN"
  else level="ERROR"; fi

  # Pick message
  eval "msgs=(\"\${${svc}_${level,,}[@]}\")"
  msg="${msgs[$((RANDOM % ${#msgs[@]}))]}"

  # Timestamp: spread over last 24h
  offset=$((RANDOM % 86400))
  ts=$((NOW - offset))
  iso=$(date -u -d "@$ts" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -r "$ts" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "2025-04-12T00:00:00Z")

  # Status code
  if [ "$level" = "ERROR" ]; then sc=$((500 + RANDOM % 4))
  elif [ "$level" = "WARN" ]; then sc=$((400 + RANDOM % 30))
  else sc=200; fi

  dur=$(echo "scale=1; $((RANDOM % 2000)) / 10" | bc 2>/dev/null || echo "$((RANDOM % 200))")
  tid=$(printf "trace-%04x%04x" $((RANDOM)) $((RANDOM)))
  uid=$(printf "user-%03d" $((RANDOM % 50)))

  BULK+='{"index":{"_index":"'"$INDEX"'"}}'$'\n'
  BULK+='{"@timestamp":"'"$iso"'","service":"'"$svc"'","level":"'"$level"'","message":"'"$msg"'","status_code":'"$sc"',"duration_ms":'"$dur"',"trace_id":"'"$tid"'","user_id":"'"$uid"'"}'$'\n'

  # Flush every 50 docs
  if [ $((i % 50)) -eq 0 ]; then
    echo -n "$i "
    curl -sf -X POST "$PROXY/_bulk" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/x-ndjson" \
      -d "$BULK" > /dev/null
    BULK=""
  fi
done

# Flush remaining
if [ -n "$BULK" ]; then
  curl -sf -X POST "$PROXY/_bulk" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/x-ndjson" \
    -d "$BULK" > /dev/null
fi
echo "OK"

# Refresh index
curl -sf -X POST "$PROXY/$INDEX/_refresh" \
  -H "Authorization: Bearer $TOKEN" > /dev/null 2>&1 || true

# Verify
echo ""
echo "=== Verification ==="
COUNT=$(curl -sf "$PROXY/$INDEX/_count" \
  -H "Authorization: Bearer $TOKEN" 2>/dev/null \
  | grep -o '"count":[0-9]*' | cut -d: -f2)
echo "Documents indexed: ${COUNT:-unknown}"

echo ""
echo "Sample query — errors in payment service:"
curl -sf "$PROXY/$INDEX/_search" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":{"bool":{"must":[{"term":{"service":"payment"}},{"term":{"level":"ERROR"}}]}},"size":3}' \
  | python3 -m json.tool 2>/dev/null | head -30 || echo "(python3 not available for pretty print)"

echo ""
echo "=== Done! 500 logs seeded across 5 services ==="
echo "Try: curl -H 'Authorization: Bearer $TOKEN' $PROXY/$INDEX/_search?q=level:ERROR"
