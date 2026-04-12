VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
BINARY := oauth4os
CLI := oauth4os-cli
ECR_REPO := 544277935543.dkr.ecr.us-west-2.amazonaws.com/oauth4os

.PHONY: build test vet lint docker push build-all release clean tidy

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/proxy
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(CLI) ./cmd/cli

test:
	go test ./internal/... -count=1 -timeout 120s

vet:
	go vet ./...

lint: vet
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed, skipping"

docker:
	docker build -t $(BINARY):$(VERSION) -t $(BINARY):latest .

push: docker
	docker tag $(BINARY):latest $(ECR_REPO):latest
	docker push $(ECR_REPO):latest

build-all:
	@for os in linux darwin; do \
		for arch in amd64 arm64; do \
			echo "Building $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
				go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-$$os-$$arch ./cmd/proxy; \
		done; \
	done

release: lint test build-all docker
	cd dist && sha256sum * > checksums.txt

clean:
	rm -rf bin/ dist/

tidy:
	@command -v go >/dev/null 2>&1 && go mod tidy || \
		docker run --rm -v $(CURDIR):/app -w /app golang:1.22-alpine go mod tidy
