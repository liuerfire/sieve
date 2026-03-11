package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCreateProvider_RequiresExpectedAPIKey(t *testing.T) {
	for _, tc := range []struct {
		name     string
		provider string
		envKey   string
	}{
		{name: "anthropic", provider: "anthropic", envKey: "ANTHROPIC_API_KEY"},
		{name: "openai", provider: "openai", envKey: "OPENAI_API_KEY"},
		{name: "gemini", provider: "gemini", envKey: "GEMINI_API_KEY"},
		{name: "qwen", provider: "qwen", envKey: "QWEN_API_KEY"},
		{name: "openrouter", provider: "openrouter", envKey: "OPENROUTER_API_KEY"},
		{name: "grok", provider: "grok", envKey: "GROK_API_KEY"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Unsetenv(tc.envKey)
			_, err := CreateProvider(Config{Provider: tc.provider, Model: "test-model"})
			if err == nil {
				t.Fatal("expected missing API key to fail")
			}
		})
	}
}

func TestStaticProvider_GradeAndSummarize(t *testing.T) {
	provider := StaticProvider{
		GradeResults: []GradeResult{{GUID: "g1", Level: "critical", Reason: "fit"}},
		SummaryResult: SummaryResult{
			GUID:        "g1",
			Title:       "summary title",
			Description: "<p>summary</p>",
			Rejected:    false,
		},
	}

	grades, err := provider.Grade(context.Background(), GradeRequest{})
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if len(grades) != 1 || grades[0].Level != "critical" {
		t.Fatalf("unexpected grades: %#v", grades)
	}

	summary, err := provider.Summarize(context.Background(), SummaryRequest{})
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if summary.Title != "summary title" {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestQwenProvider_GradeUsesChatCompletions(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("expected /chat/completions path, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-qwen-key" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if err := json.Unmarshal(data, &requestBody); err != nil {
			t.Fatalf("Unmarshal request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"choices": [{
				"message": {
					"content": "{\"items\":[{\"guid\":\"g1\",\"level\":\"critical\",\"reason\":\"fit\"}]}"
				}
			}]
		}`)
	}))
	defer server.Close()

	t.Setenv("QWEN_API_KEY", "test-qwen-key")
	provider, err := CreateProvider(Config{
		Provider: "qwen",
		Model:    "qwen-plus",
		BaseURL:  server.URL,
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	results, err := provider.Grade(context.Background(), GradeRequest{
		Items: []GradeItem{{GUID: "g1", Title: "Title", Meta: "Meta"}},
	})
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if len(results) != 1 || results[0].GUID != "g1" || results[0].Level != "critical" {
		t.Fatalf("unexpected grade results: %#v", results)
	}
	if requestBody["model"] != "qwen-plus" {
		t.Fatalf("unexpected model %#v", requestBody["model"])
	}
	if _, ok := requestBody["response_format"]; !ok {
		t.Fatal("expected response_format in request body")
	}
}

func TestQwenProvider_SummarizeUsesChatCompletions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("expected /chat/completions path, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-qwen-key" {
			t.Fatalf("unexpected authorization header %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"choices": [{
				"message": {
					"content": "{\"guid\":\"g1\",\"title\":\"Summary title\",\"description\":\"<p>summary</p>\",\"rejected\":false}"
				}
			}]
		}`)
	}))
	defer server.Close()

	t.Setenv("QWEN_API_KEY", "test-qwen-key")
	provider, err := CreateProvider(Config{
		Provider: "qwen",
		Model:    "qwen-plus",
		BaseURL:  server.URL,
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	result, err := provider.Summarize(context.Background(), SummaryRequest{
		GUID:              "g1",
		Title:             "Title",
		Description:       "Description",
		PreferredLanguage: "en",
	})
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if result.GUID != "g1" || result.Title != "Summary title" {
		t.Fatalf("unexpected summary result: %#v", result)
	}
}
