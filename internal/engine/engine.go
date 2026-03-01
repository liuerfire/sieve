// Package engine orchestrates RSS feed fetching, AI filtering, and report generation.
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugin"
	"github.com/liuerfire/sieve/internal/rss"
	"github.com/liuerfire/sieve/internal/storage"
)

// Default concurrency and rate limiting configuration.
const (
	defaultAIRateLimit      = 500 * time.Millisecond // Time between AI requests
	defaultAIBurstLimit     = 5                      // Max burst AI requests
	defaultAIMaxConcurrency = 5                      // Max concurrent AI requests
)

type ProgressEvent struct {
	Type    string // "source_start", "source_done", "item_start", "item_done", "gen_start", "gen_done"
	Source  string
	Item    string
	Message string
	Level   string
	Count   int
	Total   int
}

type SourceError struct {
	Name  string
	URL   string
	Error error
}

type EngineResult struct {
	SourcesProcessed  int
	SourcesFailed     []SourceError
	ItemsProcessed    int
	ItemsHighInterest int
}

type Classifier interface {
	Classify(ctx context.Context, cfg *config.AIConfig, title, desc, rules, lang string) (string, string, string, error)
	Summarize(ctx context.Context, cfg *config.AIConfig, title, desc, lang string) (string, error)
}

type Engine struct {
	cfg        *config.Config
	storage    *storage.Storage
	ai         Classifier
	OnProgress func(ProgressEvent)
}

func NewEngine(cfg *config.Config, s *storage.Storage, a Classifier) *Engine {
	return &Engine{
		cfg:     cfg,
		storage: s,
		ai:      a,
	}
}

func (e *Engine) report(ev ProgressEvent) {
	if e.OnProgress != nil {
		e.OnProgress(ev)
	}
}

func (e *Engine) resolveAIConfig(src config.SourceConfig) *config.AIConfig {
	return config.ResolveAIConfig(e.cfg.Global.AI, src.AI)
}

func (e *Engine) Run(ctx context.Context) (*EngineResult, error) {
	parentCtx := ctx
	g, ctx := errgroup.WithContext(ctx)

	result := &EngineResult{}
	var mu sync.Mutex

	// AI Rate Limiter: Use config values with defaults
	rateLimit := defaultAIRateLimit
	if e.cfg.Global.AITimeBetweenRequests > 0 {
		rateLimit = time.Duration(e.cfg.Global.AITimeBetweenRequests) * time.Millisecond
	}
	burstLimit := defaultAIBurstLimit
	if e.cfg.Global.AIBurstLimit > 0 {
		burstLimit = e.cfg.Global.AIBurstLimit
	}
	limiter := rate.NewLimiter(rate.Every(rateLimit), burstLimit)

	// AI Semaphore: Use config value with default
	maxConcurrency := defaultAIMaxConcurrency
	if e.cfg.Global.AIMaxConcurrency > 0 {
		maxConcurrency = e.cfg.Global.AIMaxConcurrency
	}
	aiSem := make(chan struct{}, maxConcurrency)

	// Process each source in parallel
	for _, src := range e.cfg.Sources {
		src := src
		g.Go(func() error {
			e.report(ProgressEvent{Type: "source_start", Source: src.Name})
			items, err := rss.FetchItems(ctx, src.URL, src.Name)
			if err != nil {
				e.report(ProgressEvent{Type: "source_done", Source: src.Name, Message: fmt.Sprintf("Error fetching items: %v", err)})
				slog.Error("Error fetching items", "source", src.Name, "url", src.URL, "err", err)
				mu.Lock()
				result.SourcesFailed = append(result.SourcesFailed, SourceError{Name: src.Name, URL: src.URL, Error: err})
				mu.Unlock()
				return nil // continue with other sources
			}

			mu.Lock()
			result.SourcesProcessed++
			mu.Unlock()

			rules := config.BuildRulesString(e.cfg.Global, src)

			for i, item := range items {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					e.report(ProgressEvent{Type: "item_start", Source: src.Name, Item: item.Title, Count: i + 1, Total: len(items)})

					// Backpressure: Wait for limiter and acquire semaphore
					if err := limiter.Wait(ctx); err != nil {
						return err
					}

					aiSem <- struct{}{}
					err := e.processItem(ctx, src, item, rules)
					<-aiSem // Release semaphore

					if err != nil {
						slog.Error("Error processing item", "source", src.Name, "title", item.Title, "err", err)
						continue
					}

					mu.Lock()
					result.ItemsProcessed++
					if item.InterestLevel == "high_interest" {
						result.ItemsHighInterest++
					}
					mu.Unlock()

					e.report(ProgressEvent{Type: "item_done", Source: src.Name, Item: item.Title, Level: item.InterestLevel, Count: i + 1, Total: len(items)})
				}
			}
			e.report(ProgressEvent{Type: "source_done", Source: src.Name, Count: len(items), Total: len(items)})
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Report failed sources
	if len(result.SourcesFailed) > 0 {
		slog.Warn("Some sources failed", "count", len(result.SourcesFailed), "failures", result.SourcesFailed)
	}

	// Generate final output
	e.report(ProgressEvent{Type: "gen_start", Message: "Generating reports"})
	if err := e.GenerateJSON(parentCtx, "index.json"); err != nil {
		return nil, err
	}
	e.report(ProgressEvent{Type: "gen_done", Message: "Reports generated"})

	return result, nil
}

func (e *Engine) processItem(ctx context.Context, src config.SourceConfig, item *storage.Item, rules string) error {
	// 1. Early Exit check
	exists, err := e.storage.Exists(ctx, item.ID)
	if err != nil {
		return fmt.Errorf("check exists: %w", err)
	}
	if exists {
		return nil
	}

	// 2. Run initial plugins (e.g., fetch_content)
	for _, pluginName := range src.Plugins {
		p, err := plugin.Get(pluginName)
		if err != nil {
			slog.Warn("Plugin not found", "name", pluginName)
			continue
		}
		item, err = p.Execute(ctx, item)
		if err != nil {
			return fmt.Errorf("plugin %s failed: %w", pluginName, err)
		}
	}

	// 3. Resolve AI settings
	aiCfg := e.resolveAIConfig(src)

	// 4. Phase 1: Initial Classification (Title + RSS Description)
	thought1, level1, reason1, err := e.ai.Classify(ctx, aiCfg, item.Title, item.Description, rules, e.cfg.Global.PreferredLanguage)
	if err != nil {
		slog.Warn("AI initial classification failed", "title", item.Title, "err", err)
		level1 = "uninterested"
		reason1 = fmt.Sprintf("AI initial classification failed: %v", err)
	}
	item.Thought = thought1

	// 5. Conditional Deep Processing (Summarization + Phase 2 Classification)
	if src.Summarize && (level1 == "high_interest" || level1 == "interest") {
		// Determine best content for summarization
		content := item.Content
		if len(content) < 100 {
			content = item.Description
		}

		// AI Summarize
		summary, err := e.ai.Summarize(ctx, aiCfg, item.Title, content, e.cfg.Global.PreferredLanguage)
		if err != nil {
			slog.Warn("AI summarization failed", "title", item.Title, "err", err)
			item.InterestLevel = level1
			item.Reason = reason1
		} else {
			item.Summary = summary
			// Phase 2: Final Classification based on AI Summary
			thought2, level2, reason2, err := e.ai.Classify(ctx, aiCfg, item.Title, summary, rules, e.cfg.Global.PreferredLanguage)
			if err != nil {
				slog.Warn("AI final classification failed", "title", item.Title, "err", err)
				item.InterestLevel = level1
				item.Reason = reason1
			} else {
				item.Thought = thought2
				item.InterestLevel = level2
				item.Reason = reason2
			}
		}
	} else {
		item.InterestLevel = level1
		item.Reason = reason1
	}

	// 6. Atomic Persistence
	return e.storage.SaveItem(ctx, item)
}

type jsonItem struct {
	GUID        string `json:"guid"`
	Title       string `json:"title"`
	Link        string `json:"link"`
	PubDate     string `json:"pubDate"`
	Description string `json:"description"`
}

type jsonReport struct {
	SourceName  string     `json:"source_name"`
	SourceURL   string     `json:"source_url"`
	SourceTitle string     `json:"source_title,omitempty"`
	TotalItems  int        `json:"total_items"`
	Items       []jsonItem `json:"items"`
}

func (e *Engine) GenerateJSON(ctx context.Context, outputPath string) error {
	report := jsonReport{
		SourceName:  "Sieve",
		SourceURL:   "https://github.com/liuerfire/sieve",
		SourceTitle: "Sieve Aggregated Report",
		Items:       make([]jsonItem, 0),
	}

	for it, err := range e.storage.AllItems(ctx) {
		if err != nil {
			return fmt.Errorf("failed to get item from storage: %w", err)
		}

		title := it.Title
		switch it.InterestLevel {
		case "high_interest":
			title = "⭐⭐ " + title
		case "interest":
			title = "⭐ " + title
		}

		desc := it.Description
		if it.Summary != "" {
			desc = it.Summary
		}

		report.Items = append(report.Items, jsonItem{
			GUID:        it.ID,
			Title:       title,
			Link:        it.Link,
			PubDate:     it.PublishedAt.Format(time.RFC1123Z),
			Description: desc,
		})
	}
	report.TotalItems = len(report.Items)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	slog.Info("Successfully generated JSON report", "path", outputPath, "items", report.TotalItems)
	return nil
}

func (e *Engine) GenerateHTML(ctx context.Context, outputPath string) error {
	funcMap := template.FuncMap{
		"stars": func(level string) string {
			switch level {
			case "high_interest":
				return "⭐⭐"
			case "interest":
				return "⭐"
			default:
				return ""
			}
		},
	}

	tmpl, err := template.New("html").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	// Use structure from items-summarized.schema.json
	type htmlItem struct {
		GUID          string
		Title         string
		Link          string
		PubDate       string
		Description   template.HTML // HTML summary
		Source        string
		InterestLevel string
		Reason        string
	}

	report := struct {
		SourceName  string
		SourceURL   string
		SourceTitle string
		TotalItems  int
		GeneratedAt string
		Items       []htmlItem
	}{
		SourceName:  "Sieve",
		SourceURL:   "https://github.com/liuerfire/sieve",
		SourceTitle: "Sieve Aggregated Report",
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Items:       make([]htmlItem, 0),
	}

	for it, err := range e.storage.AllItems(ctx) {
		if err != nil {
			return fmt.Errorf("get item: %w", err)
		}
		desc := it.Description
		if it.Summary != "" {
			desc = it.Summary
		}

		report.Items = append(report.Items, htmlItem{
			GUID:          it.ID,
			Title:         it.Title,
			Link:          it.Link,
			PubDate:       it.PublishedAt.Format(time.RFC1123Z),
			Description:   template.HTML(desc),
			Source:        it.Source,
			InterestLevel: it.InterestLevel,
			Reason:        it.Reason,
		})
	}
	report.TotalItems = len(report.Items)

	if err := tmpl.Execute(f, report); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	slog.Info("Successfully generated HTML report", "path", outputPath, "items", report.TotalItems)
	return nil
}
