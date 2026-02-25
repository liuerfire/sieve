package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClassify(t *testing.T) {
	// Mock Gemini response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock Gemini response format
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"candidates": [{
				"content": {
					"parts": [{
						"text": "{\"type\": \"high_interest\", \"reason\": \"matched keywords\"}"
					}]
				}
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient(Gemini, "dummy-key", WithBaseURL(server.URL))

	level, reason, err := client.Classify(context.Background(), "Test Title", "Test Content", "High Interest Rules")
	if err != nil {
		t.Fatalf("failed to classify: %v", err)
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
		w.Write([]byte(`{
			"candidates": [{
				"content": {
					"parts": [{
						"text": "This is a summarized content."
					}]
				}
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient(Gemini, "dummy-key", WithBaseURL(server.URL))

	summary, err := client.Summarize(context.Background(), "Test Title", "Test Content", "zh")
	if err != nil {
		t.Fatalf("failed to summarize: %v", err)
	}

	if summary != "This is a summarized content." {
		t.Errorf("expected summary 'This is a summarized content.', got '%s'", summary)
	}
}
