.PHONY: help test test-race test-cover lint fmt vet build clean benchmark deps tidy check install-tools

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development commands
test: ## Run tests
	go test -v ./...

test-race: ## Run tests with race detection
	go test -v -race ./...

test-cover: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

benchmark: ## Run benchmarks
	go test -bench=. -benchmem ./...

# Code quality
lint: ## Run golangci-lint
	golangci-lint run --timeout=5m

fmt: ## Format code
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	go vet ./...

# Build commands
build: ## Build the example application
	go build -o bin/logmgr-example ./example

build-all: ## Build for all platforms
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o dist/logmgr-linux-amd64 ./example
	GOOS=linux GOARCH=arm64 go build -o dist/logmgr-linux-arm64 ./example
	GOOS=windows GOARCH=amd64 go build -o dist/logmgr-windows-amd64.exe ./example
	GOOS=darwin GOARCH=amd64 go build -o dist/logmgr-darwin-amd64 ./example
	GOOS=darwin GOARCH=arm64 go build -o dist/logmgr-darwin-arm64 ./example

# Dependency management
deps: ## Download dependencies
	go mod download

tidy: ## Tidy dependencies
	go mod tidy

# Cleanup
clean: ## Clean build artifacts
	rm -rf bin/ dist/ coverage.out coverage.html
	find . -name "*.log" -type f -delete
	find . -name "*_results.txt" -type f -delete
	find . -name "*.test" -type f -delete

# Development tools
install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Combined checks
check: fmt vet lint test-race ## Run all checks (format, vet, lint, test with race detection)

# CI commands
ci-test: ## Run CI tests
	go test -v -race -coverprofile=coverage.out ./...

ci-lint: ## Run CI linting
	golangci-lint run --timeout=5m --out-format=github-actions

ci-build: build-all ## Run CI build (all platforms) 