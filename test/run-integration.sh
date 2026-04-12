#!/bin/bash
# Run integration tests against docker-compose.test.yml
set -euo pipefail

COMPOSE_FILE="docker-compose.test.yml"

echo "=== Starting services ==="
docker compose -f "$COMPOSE_FILE" up -d --build --wait --wait-timeout 120

echo "=== Running integration tests ==="
PROXY_URL=http://localhost:8443 go test -v -count=1 -timeout 120s ./test/integration/...
EXIT=$?

echo "=== Cleaning up ==="
docker compose -f "$COMPOSE_FILE" down -v

exit $EXIT
