package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

type ReporterHTMLPlugin struct {
	plugins.BasePlugin
}

type reporterHTMLOptions struct {
	OutputPath string `json:"outputPath"`
	SourceName string `json:"sourceName,omitempty"`
	Title      string `json:"title,omitempty"`
}

type htmlPageData struct {
	Title       string
	SourceName  string
	Visible     int
	Critical    int
	Recommended int
	Optional    int
	Items       []htmlItem
}

type htmlItem struct {
	Title       string
	Link        string
	PubDate     string
	Description template.HTML
	Level       string
	Reason      string
	Badge       string
}

var reporterHTMLTemplate = template.Must(template.New("reporter-html").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{ .Title }}</title>
  <style>
    :root {
      --bg: #f4efe7;
      --panel: rgba(255, 252, 246, 0.86);
      --ink: #1e1a16;
      --muted: #6c6258;
      --line: rgba(71, 56, 38, 0.12);
      --shadow: 0 20px 50px rgba(48, 35, 22, 0.12);
      --critical: #9f2f22;
      --recommended: #b56a00;
      --optional: #2d6a4f;
      --accent: #d8c3a5;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      color: var(--ink);
      background:
        radial-gradient(circle at top left, rgba(226, 205, 173, 0.9), transparent 34%),
        radial-gradient(circle at top right, rgba(179, 208, 210, 0.7), transparent 28%),
        linear-gradient(180deg, #fbf7f1 0%, var(--bg) 100%);
      font-family: "Iowan Old Style", "Palatino Linotype", "Book Antiqua", Georgia, serif;
    }
    a { color: inherit; }
    .shell {
      width: min(1120px, calc(100vw - 32px));
      margin: 0 auto;
      padding: 36px 0 80px;
    }
    .hero {
      padding: 28px;
      border: 1px solid var(--line);
      border-radius: 28px;
      background: var(--panel);
      backdrop-filter: blur(16px);
      box-shadow: var(--shadow);
    }
    .eyebrow {
      margin: 0 0 12px;
      font-size: 12px;
      letter-spacing: 0.24em;
      text-transform: uppercase;
      color: var(--muted);
      font-family: "Avenir Next", "Segoe UI", sans-serif;
    }
    h1 {
      margin: 0;
      font-size: clamp(34px, 6vw, 68px);
      line-height: 0.96;
      letter-spacing: -0.04em;
      max-width: 12ch;
    }
    .summary {
      margin: 18px 0 0;
      color: var(--muted);
      font-size: 18px;
      max-width: 60ch;
    }
    .stats {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
      gap: 12px;
      margin-top: 24px;
    }
    .stat {
      padding: 14px 16px;
      border-radius: 18px;
      border: 1px solid var(--line);
      background: rgba(255, 255, 255, 0.52);
      font-family: "Avenir Next", "Segoe UI", sans-serif;
    }
    .stat strong {
      display: block;
      font-size: 28px;
      line-height: 1;
      margin-bottom: 6px;
    }
    .items {
      display: grid;
      gap: 18px;
      margin-top: 22px;
    }
    .card {
      padding: 24px;
      border-radius: 24px;
      border: 1px solid var(--line);
      background: var(--panel);
      box-shadow: var(--shadow);
      overflow: hidden;
    }
    .cardhead {
      display: flex;
      gap: 14px;
      justify-content: space-between;
      align-items: flex-start;
      margin-bottom: 16px;
    }
    .meta {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      align-items: center;
      margin-top: 10px;
      color: var(--muted);
      font-family: "Avenir Next", "Segoe UI", sans-serif;
      font-size: 14px;
    }
    .title {
      margin: 0;
      font-size: clamp(24px, 4vw, 34px);
      line-height: 1.04;
      letter-spacing: -0.03em;
    }
    .badge {
      padding: 8px 12px;
      border-radius: 999px;
      white-space: nowrap;
      color: white;
      font-size: 12px;
      letter-spacing: 0.14em;
      text-transform: uppercase;
      font-family: "Avenir Next", "Segoe UI", sans-serif;
    }
    .badge.critical { background: var(--critical); }
    .badge.recommended { background: var(--recommended); }
    .badge.optional { background: var(--optional); }
    .reason {
      margin: 0 0 16px;
      padding: 12px 14px;
      border-left: 3px solid var(--accent);
      background: rgba(216, 195, 165, 0.18);
      color: var(--muted);
      font-family: "Avenir Next", "Segoe UI", sans-serif;
    }
    .body {
      font-size: 17px;
      line-height: 1.75;
    }
    .body img {
      display: block;
      max-width: 100%;
      height: auto;
      margin: 18px auto;
      border-radius: 18px;
      border: 1px solid var(--line);
      box-shadow: 0 14px 30px rgba(31, 20, 10, 0.1);
    }
    .body p:first-child { margin-top: 0; }
    .empty {
      margin-top: 22px;
      padding: 36px 28px;
      border: 1px dashed var(--line);
      border-radius: 24px;
      text-align: center;
      color: var(--muted);
      background: rgba(255, 255, 255, 0.36);
      font-family: "Avenir Next", "Segoe UI", sans-serif;
    }
    @media (max-width: 720px) {
      .shell { width: min(100vw - 20px, 1120px); padding-top: 18px; }
      .hero, .card { padding: 18px; border-radius: 20px; }
      .cardhead { flex-direction: column; }
    }
  </style>
</head>
<body>
  <main class="shell">
    <section class="hero">
      <p class="eyebrow">{{ .SourceName }}</p>
      <h1>{{ .Title }}</h1>
      <p class="summary">A direct browser view of the latest visible items, with summaries and grading reasons.</p>
      <div class="stats">
        <div class="stat"><strong>{{ .Visible }}</strong><span>Visible Items</span></div>
        <div class="stat"><strong>{{ .Critical }}</strong><span>Critical</span></div>
        <div class="stat"><strong>{{ .Recommended }}</strong><span>Recommended</span></div>
        <div class="stat"><strong>{{ .Optional }}</strong><span>Optional</span></div>
      </div>
    </section>
    {{ if .Items }}
    <section class="items">
      {{ range .Items }}
      <article class="card">
        <div class="cardhead">
          <div>
            <h2 class="title">{{ .Title }}</h2>
            <div class="meta">
              {{ if .PubDate }}<span>{{ .PubDate }}</span>{{ end }}
              {{ if .Link }}<a href="{{ .Link }}" target="_blank" rel="noreferrer">Open original</a>{{ end }}
            </div>
          </div>
          <span class="badge {{ .Level }}">{{ .Badge }}</span>
        </div>
        {{ if .Reason }}<p class="reason">{{ .Reason }}</p>{{ end }}
        <div class="body">{{ .Description }}</div>
      </article>
      {{ end }}
    </section>
    {{ else }}
    <section class="empty">No visible items are available yet for this source.</section>
    {{ end }}
  </main>
</body>
</html>
`))

func (ReporterHTMLPlugin) Report(_ context.Context, items []types.FeedItem, entry config.PluginEntry, runCtx plugins.Context) error {
	var opts reporterHTMLOptions
	if err := json.Unmarshal(entry.Options, &opts); err != nil {
		return err
	}
	if opts.OutputPath == "" {
		return fmt.Errorf("reporter-html: outputPath is required")
	}

	page := buildHTMLPageData(items, opts)
	if runCtx.IsDryRun {
		return nil
	}

	var buf bytes.Buffer
	if err := reporterHTMLTemplate.Execute(&buf, page); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(opts.OutputPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(opts.OutputPath, buf.Bytes(), 0o644); err != nil {
		return err
	}
	if runCtx.Logger != nil {
		runCtx.Logger.Info("wrote html output", "source", runCtx.SourceName, "path", opts.OutputPath, "items", page.Visible)
	}
	return nil
}

func buildHTMLPageData(items []types.FeedItem, opts reporterHTMLOptions) htmlPageData {
	page := htmlPageData{
		Title:      opts.Title,
		SourceName: opts.SourceName,
		Items:      make([]htmlItem, 0, len(items)),
	}
	if page.Title == "" {
		page.Title = opts.SourceName
	}
	if page.SourceName == "" {
		page.SourceName = opts.Title
	}
	for _, item := range items {
		if item.Level == types.LevelRejected {
			continue
		}
		page.Visible++
		switch item.Level {
		case types.LevelCritical:
			page.Critical++
		case types.LevelRecommended:
			page.Recommended++
		case types.LevelOptional, types.LevelUnknown, "":
			page.Optional++
		default:
			page.Optional++
		}
		page.Items = append(page.Items, htmlItem{
			Title:       item.Title,
			Link:        item.Link,
			PubDate:     item.PubDate,
			Description: template.HTML(item.Description),
			Level:       string(normalizeHTMLLevel(item.Level)),
			Reason:      item.Reason,
			Badge:       badgeLabel(item.Level),
		})
	}
	return page
}

func normalizeHTMLLevel(level types.FeedLevel) types.FeedLevel {
	switch level {
	case types.LevelCritical, types.LevelRecommended:
		return level
	default:
		return types.LevelOptional
	}
}

func badgeLabel(level types.FeedLevel) string {
	switch level {
	case types.LevelCritical:
		return "Critical"
	case types.LevelRecommended:
		return "Recommended"
	default:
		return "Optional"
	}
}

func init() {
	plugins.Register("builtin/reporter-html", ReporterHTMLPlugin{})
}
