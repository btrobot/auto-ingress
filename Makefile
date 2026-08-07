.PHONY: build test clean lint install run help

BINARY := auto-ingress
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/nodecoda/auto-ingress/cmd.version=$(VERSION)"

help: ## Show help
	@echo "Auto Ingress - Build Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build binary
	go build $(LDFLAGS) -o $(BINARY) .

install: ## Install to $GOPATH/bin
	go install $(LDFLAGS) .

test: ## Run tests
	go test ./... -v

lint: ## Run linter
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "Install golangci-lint for full lint"

clean: ## Clean build artifacts
	go clean
	rm -f $(BINARY)

run: build ## Build and run
	./$(BINARY)

release-snapshot: ## Create a release snapshot
	goreleaser release --snapshot --clean

release: ## Create a release
	goreleaser release --clean
