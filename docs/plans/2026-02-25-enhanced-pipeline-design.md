# Design Doc: Enhanced Processing Pipeline

## 1. Objective
Improve classification accuracy, reduce redundant AI costs, and ensure high-quality content summaries for a large number of RSS feeds (90+ sources).

## 2. Architecture Overview
Transition from a single-pass processing model to a multi-stage conditional pipeline with strict idempotency and early-exit mechanisms.

### 2.1 Enhanced GUID Tracking (Early Exit)
- **Goal**: Skip articles that have already been processed and saved to the database.
- **Mechanism**: 
  - Add `Storage.Exists(id string) (bool, error)` to check the SQLite database before any AI calls.
  - Integration: Call `Exists` immediately after fetching items from the RSS feed.

### 2.2 Multi-Stage Pipeline (Two-Pass Classification)
- **Phase 1: Initial Grade (Fast Pass)**
  - Input: Title + RSS Description.
  - Prompt: `BuildClassifyPrompt`.
  - Outcome: `level1`.
- **Quality Check**:
  - If `level1` is `uninterested` or `exclude`, skip to **Phase 4**.
- **Phase 2: Content Enrichment (Fetch)**
  - Action: Fetch full content via `fetch_content` plugin.
  - Fallback: Use RSS `description` if full content is unavailable or too short (< 100 chars).
- **Phase 3: Deep Summarization & Regrade (Deep Dive)**
  - **Summarize**: AI generates HTML summary based on the best available content.
  - **Final Grade**: AI re-classifies the item based on the **generated summary**.
  - Outcome: `level2` (Final Interest Level).
- **Phase 4: Persistence**
  - Action: `Storage.SaveItem` (including Full Content, Summary, and `level2`).

## 3. Implementation Details

### 3.1 Three-Level Fallback for Content
1. **Plugin**: Attempt to scrape full text from the link.
2. **Description**: Use the RSS description field if the plugin fails/is missing.
3. **No-Summary Exit**: If both are insufficient, skip summarization and use `level1` as the final result.

### 3.2 Error Handling & Reliability
- **Atomic Operations**: An item is only marked as "processed" once it is successfully saved to the database.
- **Retry Mechanism**: Failures in network or AI calls will cause the item to be skipped in the current run but retried in the next run (since it won't exist in DB).

## 4. Success Criteria
- [ ] 0 duplicate AI calls for already-processed articles.
- [ ] Highly accurate classification by filtering out "clickbait" via Phase 3.
- [ ] No "hallucinated" summaries when content is insufficient.
- [ ] Full content stored in the database for future search/RAG features.
