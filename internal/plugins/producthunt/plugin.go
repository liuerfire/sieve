package producthunt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/liuerfire/sieve/internal/config"
	httpx "github.com/liuerfire/sieve/internal/http"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

type Plugin struct {
	plugins.BasePlugin
}

var graphqlURL = "https://api.producthunt.com/v2/api/graphql"
var now = time.Now

type collectOptions struct {
	Limit int `json:"limit"`
}

func (Plugin) Collect(ctx context.Context, entry config.PluginEntry, _ plugins.Context) (plugins.CollectResult, error) {
	token := os.Getenv("PRODUCTHUNT_API_KEY")
	if token == "" {
		return plugins.CollectResult{}, fmt.Errorf("PRODUCTHUNT_API_KEY not set")
	}
	var opts collectOptions
	_ = json.Unmarshal(entry.Options, &opts)
	if opts.Limit == 0 {
		opts.Limit = 10
	}

	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return plugins.CollectResult{}, err
	}
	current := now().In(la)
	postedAfter := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, la).UTC()
	if postedAfter.After(now().UTC()) {
		postedAfter = postedAfter.Add(-24 * time.Hour)
	}
	payload := map[string]any{
		"query": "query($postedAfter: DateTime!, $first: Int!) { posts(order: VOTES, first: $first, postedAfter: $postedAfter) { edges { node { id name tagline description url website votesCount createdAt topics { edges { node { name } } } } } } }",
		"variables": map[string]any{
			"postedAfter": postedAfter.Format(time.RFC3339),
			"first":       opts.Limit,
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewReader(body))
	if err != nil {
		return plugins.CollectResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpx.NewClient().Do(req)
	if err != nil {
		return plugins.CollectResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return plugins.CollectResult{}, fmt.Errorf("producthunt: unexpected status %d", resp.StatusCode)
	}

	var data struct {
		Data struct {
			Posts struct {
				Edges []struct {
					Node struct {
						ID          string `json:"id"`
						Name        string `json:"name"`
						Tagline     string `json:"tagline"`
						Description string `json:"description"`
						URL         string `json:"url"`
						Website     string `json:"website"`
						VotesCount  int    `json:"votesCount"`
						CreatedAt   string `json:"createdAt"`
						Topics      struct {
							Edges []struct {
								Node struct {
									Name string `json:"name"`
								} `json:"node"`
							} `json:"edges"`
						} `json:"topics"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"posts"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return plugins.CollectResult{}, err
	}

	items := make([]types.FeedItem, 0, len(data.Data.Posts.Edges))
	for i, edge := range data.Data.Posts.Edges {
		link := edge.Node.Website
		if link == "" {
			link = edge.Node.URL
		}
		link = resolveLink(link)
		topics := make([]string, 0, len(edge.Node.Topics.Edges))
		for _, topic := range edge.Node.Topics.Edges {
			topics = append(topics, topic.Node.Name)
		}
		desc := edge.Node.Tagline
		if edge.Node.Description != "" {
			desc += "\n\n" + edge.Node.Description
		}
		items = append(items, types.FeedItem{
			Title:       edge.Node.Name,
			Link:        link,
			PubDate:     edge.Node.CreatedAt,
			Description: desc,
			GUID:        edge.Node.ID,
			Extra: map[string]any{
				"votesCount": edge.Node.VotesCount,
				"rank":       i + 1,
				"topics":     strings.Join(topics, ", "),
				"phUrl":      edge.Node.URL,
			},
		}.WithDefaults())
	}
	return plugins.CollectResult{Title: "Product Hunt", Items: items}, nil
}

func resolveLink(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	query := parsed.Query()
	query.Del("ref")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func GraphqlURLForTest(next string) func() {
	prev := graphqlURL
	graphqlURL = next
	return func() {
		graphqlURL = prev
	}
}

func swapNow(fn func() time.Time) func() {
	prev := now
	now = fn
	return func() {
		now = prev
	}
}

func init() {
	plugins.Register("producthunt", Plugin{})
}
