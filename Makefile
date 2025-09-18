# Makefile for solana-validator-version-sync

# Variables
BINARY_NAME := solana-validator-version-sync
BUILD_DIR := bin
LDFLAGS := -ldflags="-s -w"
export COMPOSE_BAKE := true

# Build targets
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

# Default target
.PHONY: all
all: build

# Local development build
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go mod tidy
	@CGO_ENABLED=0 go build -mod=mod $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/solana-validator-version-sync

# Cross-platform build for all platforms
.PHONY: build-all
build-all:
	@echo "Building $(BINARY_NAME) for all platforms..."
	@echo "Debug: Current directory: $$(pwd)"
	@echo "Debug: Contents of cmd/:"
	@ls -la cmd/ || echo "cmd/ directory not found"
	@echo "Debug: Contents of cmd/solana-validator-version-sync/:"
	@ls -la cmd/solana-validator-version-sync/ || echo "cmd/solana-validator-version-sync/ directory not found"
	@mkdir -p $(BUILD_DIR)
	@go mod tidy
	@VERSION=$$(cat cmd/version.txt | tr -d '\n'); \
	for platform in $(PLATFORMS); do \
		OS=$$(echo $$platform | cut -d'/' -f1); \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		OUTPUT_NAME=$(BINARY_NAME)-$$VERSION-$$OS-$$ARCH; \
		echo "Building for $$OS/$$ARCH..."; \
		CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH go build -mod=mod $(LDFLAGS) -o $(BUILD_DIR)/$$OUTPUT_NAME ./cmd/solana-validator-version-sync; \
	done
	@echo "Generating checksums..."
	@cd $(BUILD_DIR) && \
	for binary in $(BINARY_NAME)-*; do \
		if [ -f "$$binary" ] && [[ ! "$$binary" == *.sha256 ]]; then \
			echo "Generating checksum for $$binary..."; \
			sha256sum "$$binary" > "$$binary.sha256"; \
		fi; \
	done
	@echo "Compressing binaries..."
	@cd $(BUILD_DIR) && \
	for binary in $(BINARY_NAME)-*; do \
		if [ -f "$$binary" ] && [[ ! "$$binary" == *.sha256 ]]; then \
			echo "Compressing $$binary..."; \
			gzip "$$binary"; \
		fi; \
	done
	@echo "Build complete. Compressed binaries and checksums are in $(BUILD_DIR)/"

# Docker build (linux-amd64)
.PHONY: build-docker
build-docker:
	@echo "Building $(BINARY_NAME) for Docker..."
	@mkdir -p $(BUILD_DIR)
	@go mod tidy
	@VERSION=$$(cat cmd/version.txt | tr -d '\n'); \
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=mod $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-$$VERSION-linux-amd64 ./cmd/solana-validator-version-sync

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -mod=mod -v ./...

# Local development
.PHONY: dev
dev:
	@echo "Running in development mode..."
	@go run ./cmd/solana-validator-version-sync run --config config.yml

# Docker development
.PHONY: dev-docker
dev-docker:
	@echo "Starting development environment with Docker Compose..."
	@docker compose -f docker-compose.dev.yml up --build solana-validator-version-sync mock-validator

# Stop Docker development
.PHONY: dev-docker-stop
dev-docker-stop:
	@echo "Stopping development environment..."
	@docker compose -f docker-compose.dev.yml down

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary locally"
	@echo "  build-all      - Build binaries for all platforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64)"
	@echo "  build-docker   - Build for Docker (linux-amd64)"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  dev            - Run in local development mode"
	@echo "  dev-docker     - Development with Docker Compose"
	@echo "  dev-docker-stop- Stop Docker development environment"
	@echo "  help           - Show this help"
