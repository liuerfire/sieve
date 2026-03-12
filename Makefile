BINARY_NAME=sieve
BIN_DIR=bin
CMD_DIR=github.com/liuerfire/sieve/cmd/sieve
GOIMPORTS=$(HOME)/go/bin/goimports
GOLANGCI_LINT=$(HOME)/go/bin/golangci-lint
MODULE_NAME=github.com/liuerfire/sieve

.PHONY: all build test clean fmt run lint lint-install lint-fast

all: build

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)

run: build
	$(BIN_DIR)/$(BINARY_NAME)

test:
	go test ./... -v

fmt:
	$(GOIMPORTS) -local $(MODULE_NAME) -w .

.PHONY: lint lint-install lint-fast

lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
	$(GOLANGCI_LINT) run ./...

lint-fast:
	$(GOLANGCI_LINT) run --fast ./...

clean:
	rm -rf $(BIN_DIR)
	rm -rf output
	rm -f sieve
