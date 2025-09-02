# RefreshArr Makefile

# Variables
BINARY_NAME=refresharr
MAIN_FILE=main.go
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Default target
.PHONY: all
all: build

# Build targets
.PHONY: build
build:
	@echo "Building RefreshArr..."
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME) $(MAIN_FILE)
	@echo "✅ Built $(BINARY_NAME)"

# Development targets
.PHONY: run
run:
	@echo "Running RefreshArr..."
	go run $(MAIN_FILE)

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
	rm -f $(BINARY_NAME) coverage.out coverage.html
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
	@echo "  build         - Build RefreshArr binary"
	@echo "  run           - Run RefreshArr"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run golangci-lint"
	@echo "  mod           - Tidy modules"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  version       - Show version"
	@echo "  help          - Show this help"
	@echo ""
	@echo "CLI Options:"
	@echo "  --version           Show version and exit"
	@echo "  --help              Show help and exit"
	@echo "  --dry-run           Run in dry-run mode (no changes)"
	@echo "  --service SERVICE   Use specific service: auto, sonarr, radarr"
	@echo "  --log-level LEVEL   Set log level: DEBUG, INFO, WARN, ERROR"
	@echo "  --series-ids IDS    Process specific series (comma-separated)"
	@echo "  --sonarr-url URL    Override Sonarr URL"
	@echo "  --sonarr-api-key KEY Override Sonarr API key"
	@echo "  --no-report         Disable terminal report output"
	@echo ""
	@echo "Examples:"
	@echo "  make build                            # Build binary"
	@echo "  ./refresharr --version               # Show version"
	@echo "  ./refresharr --dry-run               # Safe preview run"
	@echo "  ./refresharr --service sonarr        # Sonarr only"
	@echo "  ./refresharr --series-ids '123,456'  # Specific series"
	@echo "  SONARR_API_KEY=xyz make run          # Run with env var"# RefreshArr Makefile
