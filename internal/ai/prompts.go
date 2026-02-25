package ai

import "fmt"

const (
	// ClassifyPrompt is the template for categorizing news items.
	ClassifyPrompt = `# Task: Content Classification
Classify the news item based on user interest configurations, considering both title and summary.

## Classification Rules
1. **Understand User Interests**:
   - Levels: high_interest, interest, uninterested, exclude.
   - Source-specific configurations have higher priority than global ones.
2. **Contextual Judgment**:
   - Focus on the **primary theme** of the article. If an article mentions a high-interest topic but the core content is irrelevant, lower its rating.
   - Avoid simple keyword matching; you must understand the actual intent and depth of the article.

## User Configuration
Rules: %s

## Input Data
Item Title: %s
Item Content: %s

## Output Format
Return ONLY a JSON object:
{
  "type": "high_interest" | "interest" | "uninterested" | "exclude",
  "reason": "A brief explanation in the user's preferred language, e.g., 'Go 1.25 concurrency optimizations, related to Backend Engineering'."
}
`

	// SummarizePrompt is the template for summarizing news items.
	SummarizePrompt = `# Role
You are a professional content summarization expert.

# Task
Deeply understand the article and generate a structured summary.

## Language Requirements
- Preferred Language: %s
- When a technical term or proper noun appears for the first time, include its original name in parentheses if necessary, e.g., "Firefox (火狐浏览器)".

## Output Specifications (HTML Format)
1. **Core Summary**: Relaxed, conversational tone, approx 300-500 characters/words. Use <p>, <ul>, <li> tags.
2. **"Did You Know?" (Optional)**: If the content involves niche knowledge or complex background, add: <div class="did-you-know"><strong>Did you know?</strong> ...</div>.
3. **Constraint**: Do NOT output any Markdown wrappers (like ` + "`" + `html ...` + "`" + `); output raw HTML code directly.

## Input Data
Item Title: %s
Item Content: %s

Please start the summary:`
)

// BuildClassifyPrompt constructs the classification prompt.
func BuildClassifyPrompt(rules, title, content string) string {
	return fmt.Sprintf(ClassifyPrompt, rules, title, content)
}

// BuildSummarizePrompt constructs the summarization prompt.
func BuildSummarizePrompt(lang, title, content string) string {
	return fmt.Sprintf(SummarizePrompt, lang, title, content)
}


