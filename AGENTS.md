# Sieve Project Context

Sieve is a config-driven CLI that fetches, filters, enriches, summarizes, and republishes RSS/news items.

## Project Overview

- **Purpose**: Automates fetching RSS/news sources, grading and summarizing items with AI, and writing filtered RSS outputs.
- **Core Technology**: Written in Go (v1.25+), using file-based outputs plus provider-backed LLM calls.
- **Key Features**:
  - JSON config format for providers, plugins, and sources.
  - Plugin pipeline: `collect -> process -> report`.
  - Content summarization in preferred language.
  - Extensible built-in and source-specific plugins.
  - File-based RSS output and GUID dedup history.

## Project Structure

- `cmd/sieve/`: CLI entry point for `sieve <source-name> [--config <path>] [--dry-run]`.
- `internal/`:
  - `config/`: Legacy config code plus the new workflow config parser.
  - `http/`, `retry/`, `output/`: Shared runtime helpers.
  - `llm/`: Provider adapter layer for the new workflow.
  - `plugin/`: Workflow plugin registry and context.
  - `plugins/`: Built-in and source-specific plugin implementations.
  - `workflow/`: Main orchestrator for the pipeline.
- `Makefile`: Build and task automation.

## Development Workflow

### Building the Project
Use the `Makefile` for building and testing:
```bash
make build   # Builds the binary into bin/sieve
make test    # Runs all tests
make fmt     # Formats code using goimports
make clean   # Cleans up build artifacts, caches, and output files
```

### Running the Application
The application requires an AI provider API key:
```bash
export OPENAI_API_KEY=your_key

./bin/sieve hacker-news --config config.json
```

### Configuration
Configuration uses a JSON file:
- top-level `llm`
- top-level `plugins`
- `sources[]` with string or object plugin entries

Item levels in the rewrite runtime:
1. `critical` (⭐⭐)
2. `recommended` (⭐)
3. `optional`
4. `rejected`

## Technical Conventions

- **Language**: Go 1.25 (Strict Adherence)
  - Use `any` instead of `interface{}`.
  - Use `range over int` for count-based loops (`for i := range n`).
  - Use `iter.Seq` and `iter.Seq2` for streaming data from storage to reports to maintain O(1) memory complexity.
- **Project Architecture**:
  - **Strategy Pattern**: Decouple LLM logic into provider interfaces.
  - **Dependency Inversion**: Workflow orchestration depends on plugin/LLM abstractions, not concrete sources.
- **Concurrency & Reliability**:
  - **Worker Pools & Semaphores**: Always limit concurrent external API calls (AI, RSS) using semaphores to avoid rate limiting.
  - **Backpressure**: Use `golang.org/x/time/rate` to maintain smooth request flow and handle provider-side bottlenecks.
  - **Resilience**: Implement exponential backoff for all external network requests.
- **Storage**:
  - File-based outputs only.
  - Keep generated RSS and processed GUID history under `output/`.
- **Code Style**:
  - **Import Grouping**: Strictly separate imports into three blocks separated by a newline:
    1. Standard library
    2. Third-party libraries
    3. Internal project modules
    Use `goimports -local github.com/liuerfire/sieve` (via `make fmt`) to maintain this structure.
  - **Naming**: Prefer concise interface names (e.g., `Provider`) and descriptive enum types (e.g., `ProviderType`).
- **Performance**:
  - Keep plugin execution simple and bounded; only introduce concurrency where tests or provider limits justify it.
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
