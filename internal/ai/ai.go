// Package ai provides AI provider clients for content classification and summarization.
// It supports multiple AI backends (Gemini, Qwen) through a unified interface.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/liuerfire/sieve/internal/config"
)

// AI provider endpoints and configuration constants.
const (
	geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"
	qwenBaseURL   = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
	httpTimeout   = 30 * time.Second
)

// ProviderType identifies the AI provider backend.
type ProviderType string

const (
	Gemini ProviderType = "gemini"
	Qwen   ProviderType = "qwen"
)

type Provider interface {
	buildRequest(ctx context.Context, model, prompt string, isJSON bool) (*http.Request, error)
	parseResponse(body []byte) (string, error)
}

type Client struct {
	providers map[ProviderType]Provider
	http      *http.Client
}

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.http = httpClient
	}
}

// WithBaseURL sets a custom base URL for a provider (useful for testing).
func WithBaseURL(t ProviderType, url string) Option {
	return func(c *Client) {
		if p, ok := c.providers[t]; ok {
			if gp, ok := p.(*geminiProvider); ok {
				gp.baseURL = url
			} else if qp, ok := p.(*qwenProvider); ok {
				qp.baseURL = url
			}
		}
	}
}

func NewClient(opts ...Option) *Client {
	c := &Client{
		providers: make(map[ProviderType]Provider),
		http:      &http.Client{Timeout: httpTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) AddProvider(t ProviderType, apiKey string) {
	switch t {
	case Gemini:
		c.providers[Gemini] = &geminiProvider{baseURL: geminiBaseURL, apiKey: apiKey}
	case Qwen:
		c.providers[Qwen] = &qwenProvider{baseURL: qwenBaseURL, apiKey: apiKey}
	}
}

type classifyResponse struct {
	Thought string `json:"thought"`
	Type    string `json:"type"`
	Reason  string `json:"reason"`
}

func (c *Client) Classify(ctx context.Context, cfg *config.AIConfig, title, content, rules, lang string) (string, string, string, error) {
	prompt := BuildClassifyPrompt(rules, title, content, lang)

	aiText, err := c.callAI(ctx, cfg, prompt, true)
	if err != nil {
		return "", "", "", err
	}

	var result classifyResponse
	if err := json.Unmarshal([]byte(aiText), &result); err != nil {
		return "", "", "", fmt.Errorf("failed to parse AI JSON: %w, body: %s", err, aiText)
	}

	return result.Thought, result.Type, result.Reason, nil
}

func (c *Client) Summarize(ctx context.Context, cfg *config.AIConfig, title, content, lang string) (string, error) {
	prompt := BuildSummarizePrompt(lang, title, content)

	return c.callAI(ctx, cfg, prompt, false)
}

// ==============================================================================
// Gemini Provider
// ==============================================================================

type geminiProvider struct {
	baseURL string
	apiKey  string
}

func (p *geminiProvider) buildRequest(ctx context.Context, model, prompt string, isJSON bool) (*http.Request, error) {
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, model, p.apiKey)
	reqBody := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{"text": prompt},
				},
			},
		},
	}
	if isJSON {
		reqBody["generationConfig"] = map[string]any{
			"responseMimeType": "application/json",
		}
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (p *geminiProvider) parseResponse(body []byte) (string, error) {
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		return resp.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", fmt.Errorf("empty response from Gemini")
}

// ==============================================================================
// Qwen Provider
// ==============================================================================

type qwenProvider struct {
	baseURL string
	apiKey  string
}

func (p *qwenProvider) buildRequest(ctx context.Context, model, prompt string, isJSON bool) (*http.Request, error) {
	reqBody := map[string]any{
		"model": model,
		"input": map[string]any{
			"messages": []map[string]any{
				{"role": "user", "content": prompt},
			},
		},
		"parameters": map[string]any{
			"result_format": "message",
		},
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	return req, nil
}

func (p *qwenProvider) parseResponse(body []byte) (string, error) {
	var resp struct {
		Output struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if len(resp.Output.Choices) > 0 {
		return resp.Output.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("empty response from Qwen")
}

func (c *Client) callAI(ctx context.Context, cfg *config.AIConfig, prompt string, isJSON bool) (string, error) {
	providerType := Gemini
	model := ""
	if cfg != nil {
		if cfg.Provider != "" {
			providerType = ProviderType(cfg.Provider)
		}
		model = cfg.Model
	}

	p, ok := c.providers[providerType]
	if !ok {
		return "", fmt.Errorf("provider %s not configured", providerType)
	}

	// Retry logic with exponential backoff
	maxRetries := 3
	var lastErr error
	for i := range maxRetries {
		if i > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(i-1)) * time.Second
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := p.buildRequest(ctx, model, prompt, isJSON)
		if err != nil {
			return "", fmt.Errorf("build request: %w", err)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("do request: %w", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() // Close immediately after reading
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("AI request failed with status %d: %s", resp.StatusCode, string(body))
			if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
				continue
			}
			return "", lastErr
		}

		aiText, err := p.parseResponse(body)
		if err != nil {
			return "", fmt.Errorf("parse response: %w", err)
		}

		if isJSON {
			aiText = cleanJSONResponse(aiText)
		} else {
			aiText = strings.TrimSpace(aiText)
		}

		return aiText, nil
	}

	return "", fmt.Errorf("all retries failed: %w", lastErr)
}

// cleanJSONResponse removes markdown code blocks and trims whitespace from AI response.
func cleanJSONResponse(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
