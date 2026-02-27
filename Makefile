BINARY_NAME=sieve
BIN_DIR=bin
CMD_DIR=github.com/liuerfire/sieve/cmd/sieve
GOIMPORTS=$(HOME)/go/bin/goimports
MODULE_NAME=github.com/liuerfire/sieve

.PHONY: all build run report test clean fmt

all: build

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)

build-pgo:
	mkdir -p $(BIN_DIR)
	go build -pgo=auto -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)

run: build
	$(BIN_DIR)/$(BINARY_NAME) run

report: build
	$(BIN_DIR)/$(BINARY_NAME) report

test:
	go test ./... -v

fmt:
	$(GOIMPORTS) -local $(MODULE_NAME) -w .

pgo:
	go test -cpuprofile=default.pgo ./internal/engine -run TestEngine_Run

clean:
	rm -rf $(BIN_DIR)
	rm -f *.db
	rm -f index.json
	rm -f sieve
