.PHONY: test-e2e test-integration test-unit demo-up demo-down

# Start demo environment
demo-up:
	docker compose -f docker-compose.demo.yml up -d
	@echo "Waiting for services..."
	@sleep 15

# Stop demo environment
demo-down:
	docker compose -f docker-compose.demo.yml down -v

# Run E2E tests (requires demo-up)
test-e2e:
	bash test/e2e/run.sh

# Run Go E2E tests (requires demo-up + Go)
test-e2e-go:
	go test ./test/e2e/ -v -count=1 -timeout 120s

# Run integration tests
test-integration:
	bash test/run-integration.sh

# Run unit tests
test-unit:
	go test ./internal/... -v -count=1
