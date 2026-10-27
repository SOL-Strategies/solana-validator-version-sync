APP_NAME ?= solana-validator-version-sync
APP_VERSION ?= dev
BUILD_DIR ?= bin
GO ?= go
BUILD_OS_ARCH_LIST ?= $(shell $(GO) env GOHOSTOS)-$(shell $(GO) env GOHOSTARCH)

.DEFAULT_GOAL := build
.PHONY: all build build-all test smoke-test clean dev dev-docker dev-docker-stop help

all: build

build:
	APP_NAME="$(APP_NAME)" APP_VERSION="$(APP_VERSION)" BUILD_DIR="$(BUILD_DIR)" BUILD_OS_ARCH_LIST="$(BUILD_OS_ARCH_LIST)" PACKAGE=false GO="$(GO)" sh scripts/build.sh

build-all:
	APP_NAME="$(APP_NAME)" APP_VERSION="$(APP_VERSION)" BUILD_DIR="$(BUILD_DIR)" BUILD_OS_ARCH_LIST="linux-amd64 linux-arm64" PACKAGE=true GO="$(GO)" sh scripts/build.sh

test:
	$(GO) test -mod=readonly -race ./...

smoke-test:
	@mkdir -p "$(BUILD_DIR)/.smoke"
	APP_NAME="$(APP_NAME)" APP_VERSION="$(APP_VERSION)" GO="$(GO)" sh scripts/prepare-version.sh
	CGO_ENABLED=0 $(GO) build -mod=readonly -trimpath -ldflags='-s -w' -o "$(BUILD_DIR)/.smoke/$(APP_NAME)" ./cmd/solana-validator-version-sync
	@"$(BUILD_DIR)/.smoke/$(APP_NAME)" --version | grep -F -- "version $(APP_VERSION)"
	@"$(BUILD_DIR)/.smoke/$(APP_NAME)" --help >/dev/null

clean:
	rm -rf "$(BUILD_DIR)"

dev:
	$(GO) run ./cmd/solana-validator-version-sync run --config config.yml

dev-docker:
	docker compose -f docker-compose.dev.yml up --build solana-validator-version-sync mock-validator

dev-docker-stop:
	docker compose -f docker-compose.dev.yml down

help:
	@echo "Targets: build, build-all, test, smoke-test, clean, dev, dev-docker, dev-docker-stop"
