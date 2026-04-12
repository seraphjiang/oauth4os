VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
BINARY := oauth4os
CLI := oauth4os-cli

.PHONY: build build-all lint test docker release clean \
        test-e2e test-integration test-unit demo-up demo-down

## Build
build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/proxy
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(CLI) ./cmd/cli

build-all:
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
			echo "Building $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
				go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-$$os-$$arch$$ext ./cmd/proxy; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
				go build -ldflags="$(LDFLAGS)" -o dist/$(CLI)-$$os-$$arch$$ext ./cmd/cli; \
		done; \
	done

## Quality
lint:
	golangci-lint run ./...
	go vet ./...

test: test-unit

## Docker
docker:
	docker build -t $(BINARY):$(VERSION) .

## Release (local)
release: lint test build-all docker
	cd dist && sha256sum * > checksums.txt

## Clean
clean:
	rm -rf bin/ dist/

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
