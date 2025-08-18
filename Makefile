# RefreshArr Makefile

# Variables
BINARY_NAME=refresharr
CLI_BINARY_NAME=refresharr-cli
MAIN_FILE=main.go
CLI_MAIN_FILE=cmd/refresharr/main.go
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Default target
.PHONY: all
all: build

# Build targets
.PHONY: build
build: build-simple build-cli

.PHONY: build-simple
build-simple:
	@echo "Building simple RefreshArr..."
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME) $(MAIN_FILE)
	@echo "✅ Built $(BINARY_NAME)"

.PHONY: build-cli
build-cli:
	@echo "Building RefreshArr CLI..."
	go build -ldflags "-X main.version=$(VERSION)" -o $(CLI_BINARY_NAME) $(CLI_MAIN_FILE)
	@echo "✅ Built $(CLI_BINARY_NAME)"

# Development targets
.PHONY: run
run:
	@echo "Running RefreshArr (simple version)..."
	go run $(MAIN_FILE)

.PHONY: run-cli
run-cli:
	@echo "Running RefreshArr CLI..."
	go run $(CLI_MAIN_FILE)

.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

# Code quality
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "✅ Vet passed"

.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	golangci-lint run
	@echo "✅ Lint passed"

.PHONY: mod
mod:
	@echo "Tidying modules..."
	go mod tidy
	@echo "✅ Modules tidied"

# Utility targets
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME) $(CLI_BINARY_NAME) coverage.out coverage.html
	@echo "✅ Clean complete"

.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	go mod download
	@echo "✅ Dependencies downloaded"

.PHONY: version
version:
	@echo "Version: $(VERSION)"

# Help target
.PHONY: help
help:
	@echo "RefreshArr Makefile"
	@echo "=================="
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build both simple and CLI versions"
	@echo "  build-simple  - Build simple version (main.go)"
	@echo "  build-cli     - Build CLI version (cmd/refresharr/main.go)"
	@echo "  run          - Run simple version"
	@echo "  run-cli      - Run CLI version"
	@echo "  test         - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run golangci-lint"
	@echo "  mod          - Tidy modules"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Download dependencies"
	@echo "  version      - Show version"
	@echo "  help         - Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make build                    # Build both versions"
	@echo "  make build-cli                # Build only CLI version"
	@echo "  SONARR_API_KEY=xyz make run   # Run with environment variable"
