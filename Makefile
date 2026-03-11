BINARY_NAME=sieve
BIN_DIR=bin
CMD_DIR=github.com/liuerfire/sieve/cmd/sieve
GOIMPORTS=$(HOME)/go/bin/goimports
GOLANGCI_LINT=$(HOME)/go/bin/golangci-lint
MODULE_NAME=github.com/liuerfire/sieve
CACHE_DIR=$(abspath .cache)
GO_BUILD_CACHE=$(CACHE_DIR)/go-build
GO_MOD_CACHE=$(CACHE_DIR)/go-mod
GOLANGCI_CACHE=$(CACHE_DIR)/golangci-lint

.PHONY: all build test clean fmt run lint lint-install lint-fast

all: build

build:
	mkdir -p $(BIN_DIR)
	mkdir -p $(GO_BUILD_CACHE) $(GO_MOD_CACHE)
	GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go build -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)

run: build
	$(BIN_DIR)/$(BINARY_NAME)

test:
	mkdir -p $(GO_BUILD_CACHE) $(GO_MOD_CACHE)
	GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test ./... -v

fmt:
	$(GOIMPORTS) -local $(MODULE_NAME) -w .

.PHONY: lint lint-install lint-fast

lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
	mkdir -p $(GO_BUILD_CACHE) $(GO_MOD_CACHE) $(GOLANGCI_CACHE)
	GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_CACHE) $(GOLANGCI_LINT) run ./...

lint-fast:
	mkdir -p $(GO_BUILD_CACHE) $(GO_MOD_CACHE) $(GOLANGCI_CACHE)
	GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_CACHE) $(GOLANGCI_LINT) run --fast ./...

clean:
	rm -rf $(BIN_DIR)
	rm -rf .cache
	rm -rf output
	rm -f sieve
