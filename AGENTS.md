# Sieve Project Context

Sieve is an intelligent RSS news aggregator that uses AI to automatically filter and summarize news content based on user-defined interests.

## Project Overview

- **Purpose**: Automates the process of fetching RSS feeds, categorizing items using AI, and providing a unified Web UI for reading and management.
- **Core Technology**: Written in Go (v1.25+), utilizing SQLite for local storage, AI (Gemini or Qwen) for content analysis, and React (TypeScript) for the Web UI.
- **Key Features**:
  - 4-level interest categorization (`high_interest`, `interest`, `uninterested`, `exclude`).
  - Interactive Web UI for reading, monitoring runs, and managing config.
  - Content summarization in preferred language.
  - Extensible plugin system for fetching metadata and full content.
  - Automated scheduling via GitHub Actions.

## Project Structure

- `cmd/sieve/`: CLI entry point and command implementations (`serve`).
- `internal/`:
  - `ai/`: Clients for AI providers (Gemini, Qwen).
  - `config/`: Configuration file management (JSON).
  - `engine/`: Main orchestrator for fetching, filtering, and summarizing.
  - `server/`: HTTP server for the Web UI and API.
  - `rss/`: RSS feed fetching and parsing logic.
  - `storage/`: SQLite database operations and schema management.
  - `plugin/`: Plugin implementations for specific sources (e.g., cnBeta, Hacker News).
- `web/`: React (TypeScript) source code for the Web UI.
- `schemas/`: JSON schemas for configuration and internal data structures.
- `Makefile`: Build and task automation.
- `run.sh`: Helper script for running the application.

## Development Workflow

### Building the Project
Use the `Makefile` for building and testing:
```bash
make build   # Builds the binary into bin/sieve
make test    # Runs all tests
make fmt     # Formats code using goimports
make clean   # Cleans up build artifacts and temporary databases
```

### Running the Application
The application requires an AI provider API key:
```bash
export GEMINI_API_KEY=your_key
# OR
export QWEN_API_KEY=your_key

./bin/sieve serve --refresh-now  # Run one refresh cycle against the database
./bin/sieve serve                # Start the Web UI dashboard (default: localhost:8080)
```

### Configuration
Configuration is managed from the SQLite-backed settings and feeds tables, typically through the Web UI/API.
Interest levels:
1. `high_interest` (⭐⭐)
2. `interest` (⭐)
3. `uninterested` (Visible but no stars)
4. `exclude` (Hidden)

## Technical Conventions

- **Language**: Go 1.25 (Strict Adherence)
  - Use `any` instead of `interface{}`.
  - Use `range over int` for count-based loops (`for i := range n`).
  - Use `iter.Seq` and `iter.Seq2` for streaming data from storage to reports to maintain O(1) memory complexity.
- **Project Architecture**:
  - **Strategy Pattern**: Decouple AI logic into `Provider` interfaces to allow seamless addition of new LLM backends.
  - **Dependency Inversion**: High-level modules (Engine) must depend on abstractions (Interfaces), not concrete implementations.
- **Concurrency & Reliability**:
  - **Worker Pools & Semaphores**: Always limit concurrent external API calls (AI, RSS) using semaphores to avoid rate limiting.
  - **Backpressure**: Use `golang.org/x/time/rate` to maintain smooth request flow and handle provider-side bottlenecks.
  - **Resilience**: Implement exponential backoff for all external network requests.
- **Storage**:
  - **SQLite WAL Mode**: Enable Write-Ahead Logging for improved concurrent read/write performance.
  - **Temporary Files**: Ensure `.db-wal` and `.db-shm` are excluded from version control.
- **Code Style**:
  - **Import Grouping**: Strictly separate imports into three blocks separated by a newline:
    1. Standard library
    2. Third-party libraries
    3. Internal project modules
    Use `goimports -local github.com/liuerfire/sieve` (via `make fmt`) to maintain this structure.
  - **Naming**: Prefer concise interface names (e.g., `Provider`) and descriptive enum types (e.g., `ProviderType`).
- **Performance**:
  - Support Profile-Guided Optimization (PGO) via the `Makefile` for critical processing paths.
- **Testing**: Follow standard Go testing patterns (`_test.go` files) and ensure 100% logic coverage using Mocks for AI interfaces.
- **Agent Skills**: Use golang related skills for agent interactions.

## Code Standards

### Coding Style

Before modifying files, you must read the existing content and strictly adhere to the original code/writing style.

**Key Principle**: More detailed $\neq$ more helpful. If the existing content is concise, new additions must also be concise.

### Git Operations

**Automatic committing and pushing of code is prohibited** unless explicitly instructed by the user (e.g., "commit this," "push").

* After modifying code, wait for user instructions before committing.
* Do not proactively run `git commit` or `git push`.
* The user may need to review changes or perform other operations first.
* All commit messages must be written in English.
* Changes under `docs/plans/` are optional by default and do not need to be committed unless explicitly requested.
