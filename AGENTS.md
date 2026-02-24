# Sieve Project Context

Sieve is an intelligent RSS news aggregator that uses AI to automatically filter and summarize news content based on user-defined interests.

## Project Overview

- **Purpose**: Automates the process of fetching RSS feeds, categorizing items using AI, and generating personalized news reports (RSS/HTML).
- **Core Technology**: Written in Go (v1.25+), utilizing SQLite for local storage and AI (Gemini or Qwen) for content analysis.
- **Key Features**:
  - 4-level interest categorization (`high_interest`, `interest`, `uninterested`, `exclude`).
  - Content summarization in preferred language.
  - Extensible plugin system for fetching metadata and full content.
  - Automated scheduling via GitHub Actions.

## Project Structure

- `cmd/sieve/`: CLI entry point and command implementations (`run`, `report`).
- `internal/`:
  - `ai/`: Clients for AI providers (Gemini, Qwen).
  - `config/`: Configuration file management (JSON).
  - `engine/`: Main orchestrator for fetching, filtering, and summarizing.
  - `rss/`: RSS feed fetching and parsing logic.
  - `storage/`: SQLite database operations and schema management.
  - `plugin/`: Plugin implementations for specific sources (e.g., cnBeta, Hacker News).
- `schemas/`: JSON schemas for configuration and internal data structures.
- `Makefile`: Build and task automation.
- `run.sh`: Helper script for running the application.

## Development Workflow

### Building the Project
Use the `Makefile` for building and testing:
```bash
make build   # Builds the binary into bin/sieve
make test    # Runs all tests
make clean   # Cleans up build artifacts and temporary databases
```

### Running the Application
The application requires an AI provider API key:
```bash
export GEMINI_API_KEY=your_key
# OR
export QWEN_API_KEY=your_key

./bin/sieve run      # Run the full aggregation process
./bin/sieve report   # Generate index.json and index.html from database
```

### Configuration
Configuration is managed via `config.json`. It defines global interest rules and specific RSS sources.
Interest levels:
1. `high_interest` (⭐⭐)
2. `interest` (⭐)
3. `uninterested` (Visible but no stars)
4. `exclude` (Hidden)

## Technical Conventions

- **Language**: Go 1.25
- **CLI Framework**: Cobra
- **Database**: SQLite (using `modernc.org/sqlite` for CGO-free operation)
- **Concurrency**: 
  - Uses `golang.org/x/sync/errgroup` for parallel processing of RSS sources.
  - Context-aware operations with proper cancellation propagation.
- **AI Integration**:
  - Custom client supporting Gemini and Qwen providers.
  - Uses functional options pattern (`WithHTTPClient`, `WithBaseURL`).
  - Implements AI-driven JSON classification with markdown cleanup.
- **Error Handling**: Use `fmt.Errorf` with `%w` for error wrapping.
- **Logging**: Uses `log/slog` for structured logging.
- **Testing**: Follow standard Go testing patterns (`_test.go` files).
- **Agent Skills**: Use golang related skills for agent interactions.
