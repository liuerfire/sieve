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
	"github.com/liuerfire/sieve/internal/plugin"
	"github.com/liuerfire/sieve/internal/types"
)

type Plugin struct {
	plugin.BaseWorkflowPlugin
}

var graphqlURL = "https://api.producthunt.com/v2/api/graphql"

type collectOptions struct {
	Limit int `json:"limit"`
}

func (Plugin) Collect(ctx context.Context, entry config.WorkflowPluginEntry, _ plugin.WorkflowContext) (plugin.CollectResult, error) {
	token := os.Getenv("PRODUCTHUNT_API_KEY")
	if token == "" {
		return plugin.CollectResult{}, fmt.Errorf("PRODUCTHUNT_API_KEY not set")
	}
	var opts collectOptions
	_ = json.Unmarshal(entry.Options, &opts)
	if opts.Limit == 0 {
		opts.Limit = 10
	}

	now := time.Now().UTC()
	pstMidnight := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.UTC)
	if pstMidnight.After(now) {
		pstMidnight = pstMidnight.Add(-24 * time.Hour)
	}
	payload := map[string]any{
		"query": "query($postedAfter: DateTime!, $first: Int!) { posts(order: VOTES, first: $first, postedAfter: $postedAfter) { edges { node { id name tagline description url website votesCount createdAt topics { edges { node { name } } } } } } }",
		"variables": map[string]any{
			"postedAfter": pstMidnight.Format(time.RFC3339),
			"first":       opts.Limit,
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewReader(body))
	if err != nil {
		return plugin.CollectResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpx.NewClient().Do(req)
	if err != nil {
		return plugin.CollectResult{}, err
	}
	defer resp.Body.Close()

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
		return plugin.CollectResult{}, err
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
	return plugin.CollectResult{Title: "Product Hunt", Items: items}, nil
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

func init() {
	plugin.RegisterWorkflow("producthunt", Plugin{})
}
