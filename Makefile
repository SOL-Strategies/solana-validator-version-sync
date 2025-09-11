# Makefile for solana-validator-version-sync

# Variables
BINARY_NAME := solana-validator-version-sync
BUILD_DIR := bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev')
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

# Default target
.PHONY: all
all: build

# Build targets for different platforms
BUILD_TARGETS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64

# Development build (current platform)
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	@go mod tidy
	@go build -mod=mod $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/solana-validator-version-sync

# Build for Docker development (linux-amd64)
.PHONY: build-docker
build-docker:
	@echo "Building $(BINARY_NAME) for Docker (linux-amd64)..."
	@mkdir -p $(BUILD_DIR)
	@go mod tidy
	@GOOS=linux GOARCH=amd64 go build -mod=mod $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/solana-validator-version-sync

# Build for all release platforms
.PHONY: build-all
build-all:
	@echo "Building $(BINARY_NAME) for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@go mod tidy
	@for target in $(BUILD_TARGETS); do \
		echo "Building for $$target..."; \
		GOOS=$$(echo $$target | cut -d'-' -f1) GOARCH=$$(echo $$target | cut -d'-' -f2) go build -mod=mod $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-$$target$$(if [ "$$target" = "windows-amd64" ]; then echo ".exe"; fi) ./cmd/solana-validator-version-sync; \
	done

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Run linting
.PHONY: lint
lint:
	@echo "Running linter..."
	@go vet ./...
	@golangci-lint run

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Run the program with demo config
.PHONY: run-demo
run-demo: build
	@echo "Running with demo configuration..."
	@./$(BUILD_DIR)/$(BINARY_NAME) run --config demo/config.yaml

# Start mock server
.PHONY: mock-server
mock-server:
	@echo "Starting mock server..."
	@cd mock-server && go run main.go config.yaml

# Run the program in development mode
.PHONY: dev
dev:
	@echo "Running in development mode..."
	@go run ./cmd/solana-validator-version-sync run --config demo/config.yaml

# Development with hot reload using Docker Compose (foreground with logs)
.PHONY: dev-docker
dev-docker:
	@echo "Starting development environment with Docker Compose..."
	@docker compose -f docker-compose.dev.yml up --build

# Stop Docker Compose development environment
.PHONY: dev-docker-stop
dev-docker-stop:
	@echo "Stopping development environment..."
	@docker compose -f docker-compose.dev.yml down

# Development setup (local)
.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	@go mod download
	@go mod tidy
	@go install github.com/air-verse/air@latest
	@echo "Development environment ready! Run 'air' to start with hot reloading."

# Docker build
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):$(VERSION) .

# Docker run
.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 -v $(PWD)/demo/config.yaml:/app/config.yaml $(BINARY_NAME):$(VERSION) run --config /app/config.yaml

# Install the binary to GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install ./cmd/solana-validator-version-sync

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  build-docker   - Build for Docker (linux-amd64)"
	@echo "  build-all      - Build for all platforms"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  deps           - Download dependencies"
	@echo "  run-demo       - Run with demo configuration"
	@echo "  mock-server    - Start mock server"
	@echo "  dev            - Run in development mode"
	@echo "  dev-docker     - Development with Docker Compose (hot reload)"
	@echo "  dev-docker-stop- Stop Docker Compose development environment"
	@echo "  dev-setup      - Setup development environment"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  install        - Install the binary"
	@echo "  help           - Show this help"
