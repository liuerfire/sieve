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

.PHONY: all build run test clean fmt

all: build

.PHONY: web-build
web-build:
	cd web && npm install && npm run build
	mkdir -p internal/server/dist
	cp -r dist/* internal/server/dist/

build: web-build
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)

build-pgo:
	mkdir -p $(BIN_DIR)
	go build -pgo=auto -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)

run: build
	$(BIN_DIR)/$(BINARY_NAME) run

test:
	go test ./... -v

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

pgo:
	go test -cpuprofile=default.pgo ./internal/engine -run TestEngine_Run

clean:
	rm -rf $(BIN_DIR)
	rm -f *.db
	rm -f sieve
