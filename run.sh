#!/bin/bash

# Sieve execution script
# Automatically compiles and runs the Sieve aggregator

# 1. Load .env file if it exists
if [ -f .env ]; then
    echo "Loading .env file..."
    export $(grep -v '^#' .env | xargs)
fi

# 2. Compile the project
echo "Compiling Sieve..."
make build
if [ $? -ne 0 ]; then
    echo "Error: Compilation failed."
    exit 1
fi

# 3. Run Sieve
BINARY="./bin/sieve"

# If no arguments are provided, start the Web server against the default database.
if [ $# -eq 0 ]; then
    DB_FILE="sieve.db"
    echo "Starting Sieve Web UI (using default database: $DB_FILE)..."
    $BINARY serve --db "$DB_FILE"
else
    # Forward all arguments to sieve
    echo "Executing: $BINARY $@"
    $BINARY "$@"
fi

if [ $? -eq 0 ]; then
    echo "Sieve execution completed successfully."
else
    echo "Error: Sieve failed during execution."
    exit 1
fi
