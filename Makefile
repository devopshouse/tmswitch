BINARY     := tmswitch
MODULE     := github.com/devopshouse/tmswitch
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS    := -ldflags "-X main.appVersion=$(VERSION) -X main.appCommit=$(COMMIT) -s -w"

GOOS       ?= $(shell go env GOOS)
GOARCH     ?= $(shell go env GOARCH)
BUILD_DIR  := dist

GOLANGCI_LINT_VERSION ?= latest
GOLANGCI_LINT         := $(shell go env GOPATH)/bin/golangci-lint
LINT_CONFIG           := .github/linters/.golangci.yml

.PHONY: all build install test test-short fmt fmt-check vet lint check install-hooks tidy clean release help

all: build

## build: compile binary for the current platform
build:
	go build $(LDFLAGS) -o $(BINARY) .

## install: build and install to GOPATH/bin
install:
	go install $(LDFLAGS) .

## test: run all tests with verbose output
test:
	go test ./... -v -count=1

## test-short: run all tests (quiet)
test-short:
	go test ./...

## fmt: format source code in place
fmt:
	gofmt -w .

## fmt-check: verify formatting without modifying files (used in CI check)
fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not gofmt-formatted:"; \
		echo "$$unformatted"; \
		echo "Run 'make fmt' to fix."; \
		exit 1; \
	fi
	@echo "✓ fmt"

## vet: run go vet
vet:
	@go vet ./... && echo "✓ vet"

## lint: run golangci-lint using the same config as CI (auto-installs if missing)
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --config $(LINT_CONFIG) ./...
	@echo "✓ lint"

$(GOLANGCI_LINT):
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

## check: run all validations locally — mirrors CI (fmt, vet, test, lint)
check: fmt-check vet test-short lint
	@echo ""
	@echo "✓ All checks passed."

## install-hooks: install a git pre-commit hook that runs 'make check'
install-hooks:
	@printf '#!/bin/sh\nset -e\nmake check\n' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed — 'make check' will run before every commit."

## tidy: tidy go.mod and go.sum
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)

## release: cross-compile binaries for all supported platforms into dist/
release: clean
	@mkdir -p $(BUILD_DIR)
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)_linux_amd64   .
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)_linux_arm64   .
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)_darwin_amd64  .
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)_darwin_arm64  .
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)_windows_amd64.exe .
	@echo "Binaries written to $(BUILD_DIR)/"

## help: show this help message
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
