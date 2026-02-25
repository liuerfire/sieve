BINARY_NAME=sieve
BIN_DIR=bin
CMD_DIR=github.com/liuerfire/sieve/cmd/sieve

.PHONY: all build run report test clean

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

pgo:
	go test -cpuprofile=default.pgo ./internal/engine -run TestEngine_Run

clean:
	rm -rf $(BIN_DIR)
	rm -f *.db
	rm -f index.json
	rm -f sieve
