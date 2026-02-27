package ai

import "fmt"

const (
	// ClassifyPrompt is the template for categorizing news items.
	ClassifyPrompt = `# Task: Content Classification
You are an intelligent news filter. Your goal is to accurately categorize news items based on specific user interest rules.

## Classification Levels
1. **high_interest** (⭐⭐): Content that perfectly matches the user's core interests or professional focus.
2. **interest** (⭐): Content that is generally relevant or interesting to the user.
3. **uninterested**: Content that doesn't match specific interests but isn't explicitly excluded.
4. **exclude**: Content that should be completely hidden (e.g., ads, off-topic, or explicitly blocked keywords).

## Strategic Guidelines
- **Priority**: Source-specific rules > Global rules.
- **Deep Understanding**: Don't just match keywords. Analyze the *intent* and *substance* of the article.
- **Context matters**: A tech article about "Apple" might be high interest, but a news item about an "Apple orchard" is likely uninterested unless specified.

## User Rules
%s

## Language Preference
User preferred language for the "reason" field: %s

## Input Data
- **Title**: %s
- **Content/Summary**: %s

## Output Format
Return ONLY a JSON object:
{
  "thought": "Internal reasoning (in English) about why this item fits the chosen category.",
  "type": "high_interest" | "interest" | "uninterested" | "exclude",
  "reason": "A concise, user-facing explanation in the user's preferred language (e.g., 'Go 1.25 runtime optimizations, highly relevant to backend performance')."
}
`

	// SummarizePrompt is the template for summarizing news items.
	SummarizePrompt = `# Task: Professional Content Summarization
Generate a structured, insightful summary of the following news article.

## Output Requirements (HTML)
1. **TL;DR (1-2 sentences)**: A bolded summary of the most critical takeaway. Wrap in <p><strong>TL;DR:</strong> ...</p>.
2. **Key Highlights**: Use a <ul> with 3-5 <li> items covering the core facts, technical details, or unique insights.
3. **Context & Impact**: A brief paragraph (<p>) explaining *why* this matters or the broader context.
4. **"Did You Know?" (Optional)**: If there's an interesting background fact, use <div class="did-you-know"><strong>💡 Did you know?</strong> ...</div>.

## Style & Language
- **Language**: %s
- **Tone**: Professional, analytical, yet engaging. Avoid marketing fluff.
- **Clarity**: Use clear headers or bold text for emphasis.
- **Terminology**: For technical terms in a non-English summary, include the original English term in parentheses on first mention, e.g., "Generic (泛型)".

## Constraints
- **Format**: Raw HTML only. No Markdown wrappers.
- **Objective**: Do not hallucinate. If content is insufficient, provide a very brief summary based on what is available.

## Input Data
- **Title**: %s
- **Content**: %s

Please provide the summary now:`
)

// BuildClassifyPrompt constructs the classification prompt.
func BuildClassifyPrompt(rules, title, content, lang string) string {
	return fmt.Sprintf(ClassifyPrompt, rules, lang, title, content)
}

// BuildSummarizePrompt constructs the summarization prompt.
func BuildSummarizePrompt(lang, title, content string) string {
	return fmt.Sprintf(SummarizePrompt, lang, title, content)
}
