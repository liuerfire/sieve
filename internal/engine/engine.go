package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugin"
	"github.com/liuerfire/sieve/internal/rss"
	"github.com/liuerfire/sieve/internal/storage"
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

type Classifier interface {
	Classify(ctx context.Context, title, desc, rules string) (string, string, error)
	Summarize(ctx context.Context, title, desc, lang string) (string, error)
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

func (e *Engine) Run(ctx context.Context) error {
	parentCtx := ctx
	g, ctx := errgroup.WithContext(ctx)

	// AI Rate Limiter: Limit total requests per second to avoid hitting bursts
	// 2 requests per second, with a burst of 5
	limiter := rate.NewLimiter(rate.Every(500*time.Millisecond), 5)

	// AI Semaphore: Limit total concurrent AI requests
	const maxAIConcurrency = 5
	aiSem := make(chan struct{}, maxAIConcurrency)

	// Process each source in parallel
	for _, src := range e.cfg.Sources {
		src := src // capture range variable
		g.Go(func() error {
			e.report(ProgressEvent{Type: "source_start", Source: src.Name})
			items, err := rss.FetchItems(src.URL, src.Name)
			if err != nil {
				e.report(ProgressEvent{Type: "source_done", Source: src.Name, Message: fmt.Sprintf("Error fetching items: %v", err)})
				slog.Error("Error fetching items", "source", src.Name, "url", src.URL, "err", err)
				return nil // continue with other sources
			}

			high := merge(e.cfg.Global.HighInterest, src.HighInterest)
			interest := merge(e.cfg.Global.Interest, src.Interest)
			uninterested := merge(e.cfg.Global.Uninterested, src.Uninterested)
			exclude := merge(e.cfg.Global.Exclude, src.Exclude)

			rules := fmt.Sprintf("High: %s, Interest: %s, Uninterested: %s, Exclude: %s",
				high, interest, uninterested, exclude)

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
					e.report(ProgressEvent{Type: "item_done", Source: src.Name, Item: item.Title, Level: item.InterestLevel, Count: i + 1, Total: len(items)})
				}
			}
			e.report(ProgressEvent{Type: "source_done", Source: src.Name, Count: len(items), Total: len(items)})
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	// Generate final output
	e.report(ProgressEvent{Type: "gen_start", Message: "Generating reports"})
	if err := e.GenerateJSON(parentCtx, "index.json"); err != nil {
		return err
	}
	e.report(ProgressEvent{Type: "gen_done", Message: "Reports generated"})
	return nil
}

func (e *Engine) processItem(ctx context.Context, src config.SourceConfig, item *storage.Item, rules string) error {
	// Run plugins
	for _, pluginName := range src.Plugins {
		p, err := plugin.Get(pluginName)
		if err != nil {
			slog.Warn("Plugin not found", "name", pluginName)
			continue
		}
		item, err = p.Execute(item)
		if err != nil {
			return fmt.Errorf("plugin %s failed: %w", pluginName, err)
		}
	}

	// Classify
	level, reason, err := e.ai.Classify(ctx, item.Title, item.Description, rules)
	if err != nil {
		slog.Warn("AI classification failed, falling back to other", "title", item.Title, "err", err)
		level = "other"
		reason = "AI classification failed"
	}
	item.InterestLevel = level
	item.Reason = reason

	// Summarize if needed
	if src.Summarize && level != "exclude" {
		summary, err := e.ai.Summarize(ctx, item.Title, item.Description, e.cfg.Global.PreferredLanguage)
		if err != nil {
			slog.Warn("AI summarization failed", "title", item.Title, "err", err)
		} else {
			item.Summary = summary
		}
	}

	// Save to storage
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

func merge(global, specific string) string {
	if specific == "" {
		return global
	}
	if global == "" {
		return specific
	}
	return global + "," + specific
}
