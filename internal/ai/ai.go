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
)

type Provider string

const (
	Gemini Provider = "gemini"
	Qwen   Provider = "qwen"
)

type Client struct {
	provider Provider
	apiKey   string
	baseURL  string
	http     *http.Client
}

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.http = httpClient
	}
}

// WithBaseURL sets a custom base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

func NewClient(provider Provider, apiKey string, opts ...Option) *Client {
	c := &Client{
		provider: provider,
		apiKey:   apiKey,
		http:     &http.Client{Timeout: 30 * time.Second},
	}

	if provider == Gemini {
		c.baseURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent"
	} else if provider == Qwen {
		c.baseURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type classifyResponse struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

func (c *Client) Classify(ctx context.Context, title, content, rules string) (string, string, error) {
	prompt := fmt.Sprintf(`Analyze the following news item and classify it based on the interest rules.
Rules: %s
Item Title: %s
Item Content: %s

Respond ONLY with a JSON object containing:
"type": (one of: "high_interest", "interest", "other", "exclude")
"reason": (a brief explanation)
`, rules, title, content)

	aiText, err := c.callAI(ctx, prompt, true)
	if err != nil {
		return "", "", err
	}

	var result classifyResponse
	if err := json.Unmarshal([]byte(aiText), &result); err != nil {
		return "", "", fmt.Errorf("failed to parse AI JSON: %w, body: %s", err, aiText)
	}

	return result.Type, result.Reason, nil
}

func (c *Client) Summarize(ctx context.Context, title, content, lang string) (string, error) {
	prompt := fmt.Sprintf(`Summarize the following news item in %s language. 
The summary should be concise and in HTML format (e.g., using <p>, <ul>, <li>).
Item Title: %s
Item Content: %s
`, lang, title, content)

	return c.callAI(ctx, prompt, false)
}

func (c *Client) callAI(ctx context.Context, prompt string, isJSON bool) (string, error) {
	var requestBody []byte
	var err error
	var url string

	if c.provider == Gemini {
		url = fmt.Sprintf("%s?key=%s", c.baseURL, c.apiKey)
		reqBody := map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]interface{}{
						{"text": prompt},
					},
				},
			},
		}
		if isJSON {
			reqBody["generationConfig"] = map[string]interface{}{
				"responseMimeType": "application/json",
			}
		}
		requestBody, err = json.Marshal(reqBody)
	} else if c.provider == Qwen {
		url = c.baseURL
		reqBody := map[string]interface{}{
			"model": "qwen-max",
			"input": map[string]interface{}{
				"messages": []map[string]interface{}{
					{"role": "user", "content": prompt},
				},
			},
			"parameters": map[string]interface{}{
				"result_format": "message",
			},
		}
		requestBody, err = json.Marshal(reqBody)
	}

	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.provider == Qwen {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var aiText string
	if c.provider == Gemini {
		var geminiResp struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal(body, &geminiResp); err != nil {
			return "", fmt.Errorf("unmarshal gemini: %w", err)
		}
		if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
			aiText = geminiResp.Candidates[0].Content.Parts[0].Text
		}
	} else if c.provider == Qwen {
		var qwenResp struct {
			Output struct {
				Choices []struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				} `json:"choices"`
			} `json:"output"`
		}
		if err := json.Unmarshal(body, &qwenResp); err != nil {
			return "", fmt.Errorf("unmarshal qwen: %w", err)
		}
		if len(qwenResp.Output.Choices) > 0 {
			aiText = qwenResp.Output.Choices[0].Message.Content
		}
	}

	aiText = strings.TrimSpace(aiText)
	if isJSON {
		// Clean up markdown code blocks if AI produced them
		aiText = strings.TrimPrefix(aiText, "```json")
		aiText = strings.TrimPrefix(aiText, "```")
		aiText = strings.TrimSuffix(aiText, "```")
		aiText = strings.TrimSpace(aiText)
	}

	return aiText, nil
}
