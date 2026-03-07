package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClassify(t *testing.T) {
	// Mock Gemini response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock Gemini response format
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{
			"candidates": [{
				"content": {
					"parts": [{
						"text": "{\"thought\": \"Matches high interest rules\", \"type\": \"high_interest\", \"reason\": \"matched keywords\"}"
					}]
				}
			}]
		}`)); err != nil {
			t.Fatalf("write mock classify response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.AddProvider(Gemini, "dummy-key")
	WithBaseURL(Gemini, server.URL)(client)

	thought, level, reason, err := client.Classify(t.Context(), nil, "Test Title", "Test Content", "High Interest Rules", "en")
	if err != nil {
		t.Fatalf("failed to classify: %v", err)
	}

	if thought != "Matches high interest rules" {
		t.Errorf("expected thought 'Matches high interest rules', got '%s'", thought)
	}
	if level != "high_interest" {
		t.Errorf("expected level 'high_interest', got '%s'", level)
	}
	if reason != "matched keywords" {
		t.Errorf("expected reason 'matched keywords', got '%s'", reason)
	}
}

func TestSummarize(t *testing.T) {
	// Mock Gemini response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{
			"candidates": [{
				"content": {
					"parts": [{
						"text": "This is a summarized content."
					}]
				}
			}]
		}`)); err != nil {
			t.Fatalf("write mock summarize response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.AddProvider(Gemini, "dummy-key")
	WithBaseURL(Gemini, server.URL)(client)

	summary, err := client.Summarize(t.Context(), nil, "Test Title", "Test Content", "zh")
	if err != nil {
		t.Fatalf("failed to summarize: %v", err)
	}

	if summary != "This is a summarized content." {
		t.Errorf("expected summary 'This is a summarized content.', got '%s'", summary)
	}
}

func TestQwenClassifyUsesChatCompletionsShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("expected /chat/completions path, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer dummy-key" {
			t.Fatalf("unexpected authorization header %q", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["model"] != "qwen-turbo" {
			t.Fatalf("unexpected model %#v", body["model"])
		}
		messages, ok := body["messages"].([]any)
		if !ok || len(messages) != 1 {
			t.Fatalf("unexpected messages payload %#v", body["messages"])
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"thought\":\"Reasoning\",\"type\":\"interest\",\"reason\":\"matched\"}"
				}
			}]
		}`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.AddProvider(Qwen, "dummy-key")
	WithBaseURL(Qwen, server.URL)(client)

	thought, level, reason, err := client.Classify(t.Context(), nil, "Test Title", "Test Content", "rules", "en")
	if err != nil {
		t.Fatalf("failed to classify with qwen: %v", err)
	}
	if thought != "Reasoning" || level != "interest" || reason != "matched" {
		t.Fatalf("unexpected classify result: %q %q %q", thought, level, reason)
	}
}

func TestQwenSummarizeUsesChatCompletionsShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("expected /chat/completions path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "summary"
				}
			}]
		}`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.AddProvider(Qwen, "dummy-key")
	WithBaseURL(Qwen, server.URL)(client)

	summary, err := client.Summarize(t.Context(), nil, "Test Title", "Test Content", "en")
	if err != nil {
		t.Fatalf("failed to summarize with qwen: %v", err)
	}
	if summary != "summary" {
		t.Fatalf("unexpected summary %q", summary)
	}
}
