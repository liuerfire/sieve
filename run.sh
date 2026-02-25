#!/bin/bash

# Sieve execution script
# Automatically compiles and runs the Sieve aggregator

# 1. Load .env file if it exists
if [ -f .env ]; then
    echo "Loading .env file..."
    export $(grep -v '^#' .env | xargs)
fi

# 2. Check for API Key environment variables
if [ -z "$GEMINI_API_KEY" ] && [ -z "$QWEN_API_KEY" ]; then
    echo "Error: GEMINI_API_KEY or QWEN_API_KEY environment variable not found."
    echo "Please set one of them first via 'export GEMINI_API_KEY=your_key_here'."
    exit 1
fi

# 3. Compile the project
echo "Compiling Sieve..."
make build
if [ $? -ne 0 ]; then
    echo "Error: Compilation failed."
    exit 1
fi

# 4. Run Sieve
BINARY="./bin/sieve"

# If no arguments are provided, use the default run command and config file
if [ $# -eq 0 ]; then
    CONFIG_FILE="config.json"
    DB_FILE="sieve.db"

    # If config.json does not exist, warn the user
    if [ ! -f "$CONFIG_FILE" ]; then
        echo "Warning: $CONFIG_FILE not found in the current directory."
        echo "Please ensure you have created a configuration file, or run manually: './run.sh run --config <path_to_config>'."
        echo "Tip: You can also run './run.sh report' to generate a local report."
        exit 1
    fi
    
    echo "Starting Sieve aggregator (using default config: $CONFIG_FILE)..."
    $BINARY run --config "$CONFIG_FILE" --db "$DB_FILE"
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
