#!/bin/bash
# Run oauth4os integration tests with Docker
set -e

echo "=== oauth4os integration tests ==="

# Start services
echo "Starting Docker services..."
docker compose up -d --build 2>/dev/null || docker-compose up -d --build 2>/dev/null

# Wait for proxy
echo "Waiting for proxy..."
for i in $(seq 1 30); do
    if curl -sf http://localhost:8443/health > /dev/null 2>&1; then
        echo "Proxy ready"
        break
    fi
    sleep 1
done

# Run tests
echo "Running tests..."
cd "$(dirname "$0")/.."
go test ./test/integration/ -v -count=1 -timeout 60s
EXIT=$?

# Cleanup
echo "Stopping Docker services..."
docker compose down 2>/dev/null || docker-compose down 2>/dev/null

exit $EXIT
