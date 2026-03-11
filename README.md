# Sieve

Sieve runs a config-driven plugin pipeline for one source at a time and writes RSS output files under `output/`.

## What It Does

- Loads a JSON config file describing providers, plugin options, and sources.
- Runs `collect -> process -> report` for a named source.
- Supports built-in plugins for RSS collection, deduplication, metadata/content fetching, LLM grading, LLM summarization, and RSS output.
- Supports source-specific plugins for cnBeta, Hacker News, Product Hunt, Zhihu, and Zaihuapd.

## Build

Go 1.25 or later is required.

```bash
make build
```

## Run

```bash
./bin/sieve hacker-news --config config.json
```

Dry run:

```bash
./bin/sieve hacker-news --config config.json --dry-run
```

## Environment Variables

Set the API key required by your configured provider or source plugin.

LLM providers:

- `ANTHROPIC_API_KEY`
- `OPENAI_API_KEY`
- `GEMINI_API_KEY`
- `QWEN_API_KEY`
- `OPENROUTER_API_KEY`
- `GROK_API_KEY`

Source plugins:

- `PRODUCTHUNT_API_KEY`

## Config Format

Sieve uses a JSON config like this:

```json
{
  "llm": {
    "provider": "qwen",
    "baseUrl": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "models": {
      "fast": "qwen-turbo",
      "balanced": "qwen-plus",
      "powerful": "qwen-max"
    }
  },
  "plugins": {
    "builtin/reporter-rss": {
      "outputPath": "output/hacker-news.xml"
    }
  },
  "sources": [
    {
      "name": "hacker-news",
      "context": "Best posts from Hacker News",
      "plugins": [
        "builtin/collect-rss",
        "builtin/fetch-meta",
        "builtin/llm-grade",
        "builtin/reporter-rss"
      ]
    }
  ]
}
```

## Output

- RSS files are written wherever `builtin/reporter-rss.outputPath` points.
- Deduplication history is stored as `output/<source>-processed.json`.

## Contributor Note

- Files under `docs/plans/` are working notes and are optional to commit.
- Do not commit `docs/plans/` changes unless explicitly requested.
