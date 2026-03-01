# Sieve

An intelligent RSS news aggregator that uses AI to automatically filter and summarize news content based on personal interests.

## Features

- 🤖 **AI Smart Filtering**: Uses AI to automatically filter content based on 4 levels of interest.
- ⭐ **Interest Grading**: High interest (⭐⭐), General interest (⭐), and other content.
- 🌐 **Deep Analysis Mode**: Optional structured summaries (output in preferred language).
- 🔌 **Plugin System**: Supports fetching web metadata, full content, Hacker News comments, etc.
- 🔧 **Flexible Configuration**: Manage all RSS sources and interest topics via JSON configuration.
- 📡 **Automated Scheduling**: Automatically executes every hour using GitHub Actions schedule.
- 📰 **Multi-Source Support**: Supports any RSS feed.
- 📡 **Free Hosting**: Use GitHub Pages to host RSS/HTML output.

## Classification Rules

AI classifies items into 4 types based on the interest topics in the configuration:

1. **High Interest** (`high_interest`) - Appears in the report with ⭐⭐.
2. **Interest** (`interest`) - Appears in the report with ⭐.
3. **Other** (`other`) - Appears in the report without stars (vague or low interest but not excluded).
4. **Exclude** (`exclude`) - Does not appear in the report.

## Quick Start

### 1. Build the Project

Ensure you have Go 1.25 or later installed:

```bash
go build -o sieve github.com/liuerfire/sieve/cmd/sieve
```

### 2. Configure AI Providers

Sieve supports Gemini and Qwen (Tongyi Qianwen) AI providers. You can provide one or both API keys. Sieve will prioritize them based on your configuration:

```bash
export GEMINI_API_KEY=your_gemini_api_key
export QWEN_API_KEY=your_qwen_api_key
```

### 3. Run Sieve

Run with the default configuration file `config.json` and database `sieve.db`:

```bash
./sieve run
```

You can also specify custom paths:

```bash
./sieve run --config my_config.json --db my_sieve.db
```

### 4. Generate Reports

You can manually generate JSON and HTML reports from the local database:

```bash
./run.sh report
```

By default, `index.json` and `index.html` will be generated. The generated JSON format strictly follows `schemas/items-summarized.schema.json`. You can customize the paths:

```bash
./run.sh report --json out.json --html index.html
```

If you only want to generate one of them, use the skip flags:

```bash
./run.sh report --skip-json
```

### Security Notice

**Never commit API keys to version control.** Use environment variables or a `.env` file (added to `.gitignore`).

## Reference Configuration

```json
{
  "$schema": "./schemas/config.schema.json",
  "global": {
    "high_interest": "major_news,programming_tools,productivity",
    "interest": "market_trends,ai_software,programming_languages,open_source,science",
    "uninterested": "industry_figures,history,infrastructure,crypto,chips,iphone,autonomous_driving",
    "exclude": "nft,cars,aviation,gaming_consoles,development_boards,biographies",
    "preferred_language": "en",
    "timeout": 5,
    "ai": {
      "provider": "gemini",
      "model": "gemini-3-pro-preview"
    }
  },
  "sources": [
    {
      "name": "cnbeta",
      "title": "cnBeta.com - Tech News",
      "url": "https://www.cnbeta.com.tw/backend.php",
      "exclude": "health_tips,entertainment_gossip",
      "plugins": ["cnbeta_fetch_content"],
      "summarize": false
    },
    {
      "name": "hacker-news",
      "url": "https://hnrss.org/best",
      "uninterested": "security,privacy",
      "exclude": "government_policy,social_news,code_golf",
      "plugins": ["fetch_meta", "fetch_content", "hn_fetch_comments"],
      "summarize": true,
      "timeout": 20
    }
  ]
}
```
