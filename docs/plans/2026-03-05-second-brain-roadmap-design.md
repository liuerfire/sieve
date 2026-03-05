# Sieve Second Brain Roadmap Design

## Goal

Turn Sieve into a reliable "second brain" for RSS by capturing the things you care about, triaging quickly, and retrieving important items later.

## User Constraints

- Primary outcome: capture things of interest.
- Input scope: RSS feeds only.
- Expected volume: under 100 items/day.

## Approach Options

1. Capture-First (recommended)
2. AI-First
3. Knowledge-Graph-First

Selected: Capture-First, because it delivers trustworthy daily value fastest under current scope.

## MVP Scope

### In scope

- Reliable RSS ingestion and deduplication.
- AI interest classification with four levels.
- Manual correction of interest labels in UI.
- Save/bookmark flow for important items.
- Fast retrieval with search and filters.
- Weekly digest from saved and high-interest items.

### Out of scope

- Manual URL/bookmark/note capture.
- Knowledge graph and chat assistant.
- Task/project integration.

## Architecture and Data Flow

1. `run` fetches RSS items, normalizes entries, and deduplicates using stable keys.
2. AI provider classifies interest and optionally summarizes.
3. Engine persists item, classification, and summary to SQLite.
4. Web UI shows triage queue prioritized by `high_interest` then `interest`.
5. User can set `saved` and override interest labels.
6. Weekly digest job reads saved and high-interest items for digest output.

## Storage Changes (SQLite)

- Add `items.saved` (boolean) and `items.saved_at` (nullable timestamp).
- Add `items.user_interest_override` (nullable enum).
- Add FTS5 index/table for full-text retrieval.
- Add `items.duplicate_of` (nullable foreign key reference) for traceability.

## API and UI Changes

### API (`internal/server`)

- Endpoint to toggle `saved`.
- Endpoint to set/clear interest override.
- Search endpoint with filters (source/date/interest/saved).
- Digest endpoint for weekly review.

### Web UI (`web/src`)

- Add save action in item card.
- Add search/filter bar.
- Add tabs/views for `Saved` and `Digest`.

## Reliability Requirements

- Keep bounded concurrency for RSS and AI calls.
- Add retry/backoff visibility for external calls.
- Enforce idempotent upsert during run cycles.

## 90-Day Roadmap

### Phase 1: Capture Reliability (2026-03-05 to 2026-03-26)

- Harden fetching, normalization, dedup, idempotent upsert.
- Add ingestion health view: failed feeds, last success, item counts.
- KPI: feed success rate >99%; duplicate rate <2%.

### Phase 2: Interest Triage UX (2026-03-27 to 2026-04-23)

- Add triage inbox ordered by interest and recency.
- Add manual interest override and persistence.
- KPI: daily triage <15 minutes; override rate decreases over time.

### Phase 3: Save + Retrieval (2026-04-24 to 2026-05-21)

- Add save workflow and Saved view.
- Add search (FTS5) and retrieval filters.
- KPI: median retrieval <10 seconds; >=20 saved high-value items.

### Phase 4: Weekly Digest + Quality Loop (2026-05-22 to 2026-06-04)

- Generate weekly digest from saved + high-interest.
- Add quality metrics and low-value source pruning loop.
- KPI: digest read completion >70%; weekly source tuning performed.

## Success Criteria

- Sieve becomes the default RSS inbox.
- Important items are captured with few misses.
- Saved knowledge is easy to find and reuse.
- Maintenance overhead stays low.

