# S9S Terminal UI Makefile

# Build variables
BINARY_NAME=s9s
BUILD_DIR=build
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILT_BY?=$(shell whoami)

# Version package path
VERSION_PKG=github.com/jontk/s9s/internal/version

# Build flags
LDFLAGS=-ldflags "\
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).Commit=$(COMMIT) \
	-X $(VERSION_PKG).Date=$(DATE) \
	-X $(VERSION_PKG).BuiltBy=$(BUILT_BY)"

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

.PHONY: all build clean test test-unit test-integration test-performance test-ssh test-streaming coverage bench lint fmt vet tidy deps install uninstall run dev version help

# Default target
all: clean deps test build

# Show version information
version:
	@echo "Version:   $(VERSION)"
	@echo "Commit:    $(COMMIT)"
	@echo "Built:     $(DATE)"
	@echo "Built by:  $(BUILT_BY)"

# Build the application
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/s9s

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
	$(GOTEST) -coverprofile=coverage.out -covermode=atomic ./internal/...
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

# Format the code with gofumpt (stricter than gofmt)
fmt:
	@echo "Formatting code..."
	@if command -v gofumpt >/dev/null 2>&1; then \
		gofumpt -w .; \
	else \
		echo "gofumpt not installed, using go fmt..."; \
		$(GOCMD) fmt ./...; \
	fi

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
	$(GOCMD) run -race ./cmd/s9s --debug

# Development helpers
.PHONY: test-quick test-verbose test-short install-tools

# Quick tests (short mode)
test-quick:
	@echo "Running quick tests..."
	$(GOTEST) -short -timeout 5m ./internal/...

# Verbose tests
test-verbose:
	@echo "Running verbose tests..."
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./internal/...

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
	go install mvdan.cc/gofumpt@latest
	@echo "Installing pre-commit..."
	@if ! command -v pre-commit >/dev/null 2>&1; then \
		echo "Installing pre-commit (requires pip)..."; \
		pip install --user pre-commit || pip3 install --user pre-commit; \
	fi
	@echo "Setting up pre-commit hooks..."
	pre-commit install

# CI/CD targets
.PHONY: ci ci-test ci-build ci-lint

# CI pipeline
ci: ci-lint ci-test ci-build

# CI test phase
ci-test:
	@echo "Running CI tests..."
	$(GOTEST) -timeout $(TEST_TIMEOUT) -race -coverprofile=coverage.out ./internal/...
	$(GOTEST) -timeout $(INTEGRATION_TEST_TIMEOUT) ./test/integration/...

# CI build phase
ci-build:
	@echo "Running CI build..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-linux ./cmd/s9s
	CGO_ENABLED=0 GOOS=darwin $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-darwin ./cmd/s9s
	CGO_ENABLED=0 GOOS=windows $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-windows.exe ./cmd/s9s

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
		goreleaser release --clean; \
	else \
		echo "goreleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi

# Create snapshot release
release-snapshot:
	@echo "Creating snapshot release..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "goreleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi

# Check if ready for release
release-check:
	@echo "Checking release readiness..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser check; \
	else \
		echo "goreleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
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

# Demo targets
.PHONY: demos demos-ci demos-full demo-overview demo-clean

# Generate demos (alias to demos-full for backwards compatibility)
demos: demos-full

# Generate demos for CI (fast, GIF only - just overview)
demos-ci: demo-overview
	@echo "CI demos complete (overview.gif generated)"
	@rm -f demos/output/*.mp4
	@echo "Removed MP4 files (CI only uses GIFs)"

# Generate all VHS demos (GIF + MP4)
demos-full: build
	@echo "Generating all VHS demos..."
	@if ! command -v vhs >/dev/null 2>&1; then \
		echo "Error: vhs is not installed. Install with: go install github.com/charmbracelet/vhs@latest"; \
		exit 1; \
	fi
	@echo "Copying binary to ./s9s for demo recording..."
	@trap 'rm -f ./s9s' EXIT; \
	cp $(BUILD_DIR)/$(BINARY_NAME) ./s9s; \
	for tape in demos/*.tape; do \
		if [ "$$tape" != "demos/common.tape" ] && [ "$$tape" != "demos/bootstrap.tape" ]; then \
			echo "Recording: $$tape"; \
			vhs "$$tape"; \
		fi; \
	done; \
	echo "All demos generated in demos/output/"

# Generate overview demo only
demo-overview: build
	@echo "Generating overview demo..."
	@if ! command -v vhs >/dev/null 2>&1; then \
		echo "Error: vhs is not installed. Install with: go install github.com/charmbracelet/vhs@latest"; \
		exit 1; \
	fi
	@echo "Copying binary to ./s9s for demo recording..."
	@trap 'rm -f ./s9s' EXIT; \
	cp $(BUILD_DIR)/$(BINARY_NAME) ./s9s; \
	vhs demos/overview.tape; \
	echo "Overview demo generated: demos/output/overview.gif"

# Clean demo outputs
demo-clean:
	@echo "Cleaning demo outputs..."
	@rm -rf demos/output/*
	@echo "Demo outputs cleaned"

# Help target
help:
	@echo "S9S Terminal UI - Available Make Targets:"
	@echo ""
	@echo "Build Targets:"
	@echo "  build         - Build the application binary"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  uninstall     - Remove binary from GOPATH/bin"
	@echo "  version       - Show build version information"
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
	@echo "  release-check - Check if project is ready for release"
	@echo ""
	@echo "Documentation Targets:"
	@echo "  docs          - Generate documentation"
	@echo "  docs-serve    - Serve documentation on localhost:6060"
	@echo ""
	@echo "Demo Targets:"
	@echo "  demos         - Generate all VHS demo recordings"
	@echo "  demo-overview - Generate overview demo only"
	@echo "  demo-clean    - Clean demo output files"
	@echo ""
	@echo "Environment Variables:"
	@echo "  VERSION       - Override version (default: git describe)"
	@echo "  SSH_INTEGRATION_TESTS=1 - Enable SSH integration tests"
	@echo "  DOCKER_SSH_TESTS=1 - Enable Docker SSH tests"
	@echo ""
	@echo "For more information, see README.md"