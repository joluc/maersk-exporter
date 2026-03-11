.PHONY: build test clean run help

# Variables
BINARY_NAME=maersk-exporter
GO=go

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	$(GO) build -o $(BINARY_NAME) ./cmd/maersk-exporter

test: ## Run tests
	$(GO) test ./... -v -short

test-coverage: ## Run tests with coverage
	$(GO) test ./... -cover

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Format code
	$(GO) fmt ./...

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)
	rm -f test.log
	rm -f exporter.log

run: build ## Build and run the exporter
	./$(BINARY_NAME)

lint: ## Run basic linting
	$(GO) vet ./...
	$(GO) fmt ./...

all: fmt vet test build ## Run fmt, vet, test, and build
