# S9S Terminal UI Makefile

# Build variables
BINARY_NAME=s9s
BUILD_DIR=build
VERSION?=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Test variables
TEST_TIMEOUT=30m
INTEGRATION_TEST_TIMEOUT=45m

.PHONY: all build clean test test-unit test-integration test-performance test-ssh test-streaming coverage bench lint fmt vet tidy deps install uninstall run dev help

# Default target
all: clean deps test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Run all tests
test: test-unit test-integration

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -timeout $(TEST_TIMEOUT) -v ./internal/...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -timeout $(INTEGRATION_TEST_TIMEOUT) -v ./test/integration/...

# Run integration tests with SSH enabled
test-ssh:
	@echo "Running SSH integration tests..."
	SSH_INTEGRATION_TESTS=1 $(GOTEST) -timeout $(INTEGRATION_TEST_TIMEOUT) -v ./test/integration/ -run SSH

# Run integration tests with Docker SSH server
test-ssh-docker:
	@echo "Running SSH integration tests with Docker..."
	DOCKER_SSH_TESTS=1 $(GOTEST) -timeout $(INTEGRATION_TEST_TIMEOUT) -v ./test/integration/ -run Docker

# Run streaming integration tests
test-streaming:
	@echo "Running streaming integration tests..."
	$(GOTEST) -timeout $(INTEGRATION_TEST_TIMEOUT) -v ./test/integration/ -run Streaming

# Run performance integration tests
test-performance:
	@echo "Running performance integration tests..."
	$(GOTEST) -timeout $(INTEGRATION_TEST_TIMEOUT) -v ./test/integration/ -run Performance

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./test/performance/... ./test/integration/...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=coverage.out -covermode=atomic ./internal/... ./test/unit/...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint the code
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

# Format the code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Vet the code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

# Tidy go modules
tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy

# Install the binary
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Uninstall the binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

# Run in development mode with race detection
dev:
	@echo "Running in development mode..."
	$(GOCMD) run -race . -debug

# Development helpers
.PHONY: test-quick test-verbose test-short install-tools

# Quick tests (short mode)
test-quick:
	@echo "Running quick tests..."
	$(GOTEST) -short -timeout 5m ./internal/... ./test/unit/...

# Verbose tests
test-verbose:
	@echo "Running verbose tests..."
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./internal/... ./test/unit/...

# Short tests only
test-short:
	@echo "Running short tests only..."
	$(GOTEST) -short ./...

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/axw/gocov/gocov@latest
	go install github.com/matm/gocov-html@latest

# CI/CD targets
.PHONY: ci ci-test ci-build ci-lint

# CI pipeline
ci: ci-lint ci-test ci-build

# CI test phase
ci-test:
	@echo "Running CI tests..."
	$(GOTEST) -timeout $(TEST_TIMEOUT) -race -coverprofile=coverage.out ./internal/... ./test/unit/...
	$(GOTEST) -timeout $(INTEGRATION_TEST_TIMEOUT) ./test/integration/...

# CI build phase
ci-build:
	@echo "Running CI build..."
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-linux .
	CGO_ENABLED=0 GOOS=darwin $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-darwin .
	CGO_ENABLED=0 GOOS=windows $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-windows.exe .

# CI lint phase
ci-lint: fmt vet lint

# Docker targets
.PHONY: docker-build docker-run docker-test

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .

# Run in Docker container
docker-run: docker-build
	@echo "Running in Docker container..."
	docker run -it --rm $(BINARY_NAME):$(VERSION)

# Test with Docker
docker-test:
	@echo "Running tests in Docker..."
	docker run --rm -v $(PWD):/app -w /app golang:1.24 make test

# Release targets
.PHONY: release release-snapshot

# Create release
release:
	@echo "Creating release..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --rm-dist; \
	else \
		echo "goreleaser not installed. Please install it first."; \
		exit 1; \
	fi

# Create snapshot release
release-snapshot:
	@echo "Creating snapshot release..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --rm-dist; \
	else \
		echo "goreleaser not installed. Please install it first."; \
		exit 1; \
	fi

# Documentation targets
.PHONY: docs docs-serve

# Generate documentation
docs:
	@echo "Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Documentation available at: http://localhost:6060/pkg/github.com/jontk/s9s/"; \
		echo "Run 'make docs-serve' to start documentation server"; \
	else \
		go install golang.org/x/tools/cmd/godoc@latest; \
	fi

# Serve documentation
docs-serve:
	@echo "Starting documentation server..."
	godoc -http=:6060

# Help target
help:
	@echo "S9S Terminal UI - Available Make Targets:"
	@echo ""
	@echo "Build Targets:"
	@echo "  build         - Build the application binary"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  uninstall     - Remove binary from GOPATH/bin"
	@echo ""
	@echo "Development Targets:"
	@echo "  run           - Build and run the application"
	@echo "  dev           - Run in development mode with race detection"
	@echo "  fmt           - Format source code"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run golangci-lint"
	@echo ""
	@echo "Test Targets:"
	@echo "  test          - Run all tests (unit + integration)"
	@echo "  test-unit     - Run unit tests only"
	@echo "  test-integration - Run integration tests"
	@echo "  test-ssh      - Run SSH integration tests"
	@echo "  test-ssh-docker - Run SSH tests with Docker server"
	@echo "  test-streaming - Run streaming integration tests"
	@echo "  test-performance - Run performance integration tests"
	@echo "  test-quick    - Run quick tests (short mode)"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  bench         - Run benchmarks"
	@echo "  coverage      - Run tests with coverage report"
	@echo ""
	@echo "Dependency Targets:"
	@echo "  deps          - Install dependencies"
	@echo "  deps-update   - Update dependencies"
	@echo "  tidy          - Tidy go modules"
	@echo "  install-tools - Install development tools"
	@echo ""
	@echo "CI/CD Targets:"
	@echo "  ci            - Run full CI pipeline"
	@echo "  ci-test       - Run CI test phase"
	@echo "  ci-build      - Run CI build phase"
	@echo "  ci-lint       - Run CI lint phase"
	@echo ""
	@echo "Docker Targets:"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run in Docker container"
	@echo "  docker-test   - Run tests in Docker"
	@echo ""
	@echo "Release Targets:"
	@echo "  release       - Create release with goreleaser"
	@echo "  release-snapshot - Create snapshot release"
	@echo ""
	@echo "Documentation Targets:"
	@echo "  docs          - Generate documentation"
	@echo "  docs-serve    - Serve documentation on localhost:6060"
	@echo ""
	@echo "Environment Variables:"
	@echo "  VERSION       - Override version (default: git describe)"
	@echo "  SSH_INTEGRATION_TESTS=1 - Enable SSH integration tests"
	@echo "  DOCKER_SSH_TESTS=1 - Enable Docker SSH tests"
	@echo ""
	@echo "For more information, see README.md"