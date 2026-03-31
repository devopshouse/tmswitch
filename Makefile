BINARY     := tmswitch
MODULE     := github.com/devopshouse/tmswitch
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS    := -ldflags "-X main.appVersion=$(VERSION) -X main.appCommit=$(COMMIT) -s -w"

GOOS       ?= $(shell go env GOOS)
GOARCH     ?= $(shell go env GOARCH)
BUILD_DIR  := dist

.PHONY: all build install test lint clean fmt vet tidy release help

all: build

## build: compile binary for the current platform
build:
	go build $(LDFLAGS) -o $(BINARY) .

## install: build and install to GOPATH/bin
install:
	go install $(LDFLAGS) .

## test: run all tests
test:
	go test ./... -v -count=1

## test-short: run tests without verbose output
test-short:
	go test ./...

## fmt: format source code
fmt:
	gofmt -w .

## vet: run go vet
vet:
	go vet ./...

## lint: run staticcheck (install with: go install honnef.co/go/tools/cmd/staticcheck@latest)
lint:
	staticcheck ./...

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
