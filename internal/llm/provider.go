package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Config struct {
	Provider string
	Model    string
	BaseURL  string
}

type GradeItem struct {
	GUID  string
	Title string
	Meta  string
}

type GradeRequest struct {
	SourceContext      string
	Context            string
	GlobalHigh         string
	GlobalInterest     string
	GlobalUninterested string
	GlobalAvoid        string
	High               string
	Interest           string
	Uninterested       string
	Avoid              string
	Items              []GradeItem
}

type GradeResult struct {
	GUID   string
	Level  string
	Reason string
}

type SummaryRequest struct {
	PreferredLanguage string
	SourceContext     string
	Context           string
	GUID              string
	Title             string
	Description       string
	Extra             map[string]any
}

type SummaryResult struct {
	GUID        string
	Title       string
	Description string
	Rejected    bool
}

type Provider interface {
	Grade(ctx context.Context, req GradeRequest) ([]GradeResult, error)
	Summarize(ctx context.Context, req SummaryRequest) (SummaryResult, error)
}

type RemoteProvider struct {
	Config Config
}

func CreateProvider(cfg Config) (Provider, error) {
	envKey, ok := providerEnvKeys[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("unsupported provider %q", cfg.Provider)
	}
	if os.Getenv(envKey) == "" {
		return nil, fmt.Errorf("%s not set", envKey)
	}
	return RemoteProvider{Config: cfg}, nil
}

var providerEnvKeys = map[string]string{
	"anthropic":  "ANTHROPIC_API_KEY",
	"openai":     "OPENAI_API_KEY",
	"gemini":     "GEMINI_API_KEY",
	"qwen":       "QWEN_API_KEY",
	"openrouter": "OPENROUTER_API_KEY",
	"grok":       "GROK_API_KEY",
}

func (p RemoteProvider) Grade(ctx context.Context, req GradeRequest) ([]GradeResult, error) {
	if p.Config.Provider != "qwen" {
		return nil, fmt.Errorf("remote grade not implemented for %s", p.Config.Provider)
	}
	type responseEnvelope struct {
		Items []GradeResult `json:"items"`
	}
	var envelope responseEnvelope
	if err := p.callQwenJSON(ctx, buildGradePrompt(req), &envelope); err != nil {
		return nil, err
	}
	return envelope.Items, nil
}

func (p RemoteProvider) Summarize(ctx context.Context, req SummaryRequest) (SummaryResult, error) {
	if p.Config.Provider != "qwen" {
		return SummaryResult{}, fmt.Errorf("remote summarize not implemented for %s", p.Config.Provider)
	}
	var result SummaryResult
	if err := p.callQwenJSON(ctx, buildSummaryPrompt(req), &result); err != nil {
		return SummaryResult{}, err
	}
	return result, nil
}

type StaticProvider struct {
	GradeResults  []GradeResult
	SummaryResult SummaryResult
	GradeErr      error
	SummaryErr    error
}

func (p StaticProvider) Grade(_ context.Context, _ GradeRequest) ([]GradeResult, error) {
	return p.GradeResults, p.GradeErr
}

func (p StaticProvider) Summarize(_ context.Context, _ SummaryRequest) (SummaryResult, error) {
	return p.SummaryResult, p.SummaryErr
}

func (p RemoteProvider) callQwenJSON(ctx context.Context, prompt string, target any) error {
	baseURL := strings.TrimRight(p.Config.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}

	payload := map[string]any{
		"model": p.Config.Model,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]any{
			"type": "json_object",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv(providerEnvKeys["qwen"]))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qwen request failed: %s", strings.TrimSpace(string(data)))
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return err
	}
	if len(completion.Choices) == 0 {
		return fmt.Errorf("qwen response contained no choices")
	}
	return json.Unmarshal([]byte(completion.Choices[0].Message.Content), target)
}

func buildGradePrompt(req GradeRequest) string {
	lines := make([]string, 0, len(req.Items)+12)
	lines = append(lines,
		"Return JSON only.",
		`Schema: {"items":[{"guid":"string","level":"critical|recommended|optional|rejected","reason":"string"}]}`,
		"Classify each item based on the provided interest rules.",
		fmt.Sprintf("Global high interest: %s", req.GlobalHigh),
		fmt.Sprintf("Global interest: %s", req.GlobalInterest),
		fmt.Sprintf("Global uninterested: %s", req.GlobalUninterested),
		fmt.Sprintf("Global avoid: %s", req.GlobalAvoid),
		fmt.Sprintf("Source high interest: %s", req.High),
		fmt.Sprintf("Source interest: %s", req.Interest),
		fmt.Sprintf("Source uninterested: %s", req.Uninterested),
		fmt.Sprintf("Source avoid: %s", req.Avoid),
		fmt.Sprintf("Source context: %s", req.SourceContext),
		fmt.Sprintf("Extra context: %s", req.Context),
	)
	for _, item := range req.Items {
		lines = append(lines, fmt.Sprintf("Item guid=%s title=%q meta=%q", item.GUID, item.Title, item.Meta))
	}
	return strings.Join(lines, "\n")
}

func buildSummaryPrompt(req SummaryRequest) string {
	extra, _ := json.Marshal(req.Extra)
	return strings.Join([]string{
		"Return JSON only.",
		`Schema: {"guid":"string","title":"string","description":"string","rejected":true|false}`,
		fmt.Sprintf("Preferred language: %s", req.PreferredLanguage),
		fmt.Sprintf("Source context: %s", req.SourceContext),
		fmt.Sprintf("Extra context: %s", req.Context),
		fmt.Sprintf("GUID: %s", req.GUID),
		fmt.Sprintf("Title: %s", req.Title),
		fmt.Sprintf("Description: %s", req.Description),
		fmt.Sprintf("Extra: %s", string(extra)),
	}, "\n")
}
