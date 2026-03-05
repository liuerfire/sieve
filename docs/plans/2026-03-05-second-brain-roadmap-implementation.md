# Sieve Second Brain (RSS Capture) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Ship a reliable RSS-only second-brain flow in Sieve: capture, triage, save, search, and weekly digest.

**Architecture:** Extend existing `storage -> engine -> server -> web` flow with additive schema/API/UI changes. Preserve current run pipeline and introduce idempotent persistence, user overrides, saved state, and digest/query endpoints. Deliver in small TDD increments with verification at each step.

**Tech Stack:** Go 1.25, modernc SQLite (WAL + FTS5), net/http, React 18 + TypeScript + Vite.

---

### Task 1: Add storage fields for saved state and user override

**Files:**
- Modify: `internal/storage/storage.go`
- Modify: `internal/storage/storage_test.go`

**Step 1: Write failing test**

```go
func TestSaveItem_PersistsSavedAndOverride(t *testing.T) {
	// save item with Saved=true and UserInterestOverride="interest"
	// fetch via GetItems and assert round-trip
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/storage -run TestSaveItem_PersistsSavedAndOverride -v`  
Expected: FAIL (missing fields / scan mismatch).

**Step 3: Write minimal implementation**

```go
type Item struct {
	// existing fields...
	Saved                bool
	SavedAt              *time.Time
	UserInterestOverride *string
}
```

Update schema, insert/upsert, scan targets, and update SQL column lists in `AllItems` and `GetItems`.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/storage -run TestSaveItem_PersistsSavedAndOverride -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/storage/storage.go internal/storage/storage_test.go
git commit -m "feat(storage): persist saved state and interest override"
```

### Task 2: Add duplicate tracking and idempotent upsert contract

**Files:**
- Modify: `internal/storage/storage.go`
- Modify: `internal/storage/storage_test.go`

**Step 1: Write failing test**

```go
func TestSaveItem_DuplicateOfRoundTrip(t *testing.T) {
	// save item with DuplicateOf set and verify it can be read back
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/storage -run TestSaveItem_DuplicateOfRoundTrip -v`  
Expected: FAIL.

**Step 3: Write minimal implementation**

Add `DuplicateOf *string` to `storage.Item`, include DB column `duplicate_of`, insert/upsert, and row scans.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/storage -run TestSaveItem_DuplicateOfRoundTrip -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/storage/storage.go internal/storage/storage_test.go
git commit -m "feat(storage): add duplicate tracking metadata"
```

### Task 3: Add SQLite FTS5 search support in storage

**Files:**
- Modify: `internal/storage/storage.go`
- Modify: `internal/storage/storage_test.go`

**Step 1: Write failing test**

```go
func TestSearchItems_FTS5(t *testing.T) {
	// save two items with different keywords
	// search one keyword and expect one match
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/storage -run TestSearchItems_FTS5 -v`  
Expected: FAIL (method missing).

**Step 3: Write minimal implementation**

Add:
- FTS5 virtual table initialization.
- FTS synchronization on save (insert/update).
- `SearchItems(ctx, q string, limit int, filters SearchFilters) ([]*Item, error)`.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/storage -run TestSearchItems_FTS5 -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/storage/storage.go internal/storage/storage_test.go
git commit -m "feat(storage): implement FTS5 search with filters"
```

### Task 4: Add API support for save toggle and interest override

**Files:**
- Modify: `internal/server/server.go`
- Modify: `internal/server/server_test.go`
- Modify: `internal/storage/storage.go`

**Step 1: Write failing server tests**

```go
func TestHandleUpdateItem_Patch_Saved(t *testing.T) {}
func TestHandleUpdateItem_Patch_UserInterestOverride(t *testing.T) {}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/server -run TestHandleUpdateItem_Patch -v`  
Expected: FAIL.

**Step 3: Write minimal implementation**

Extend PATCH payload:

```go
var req struct {
	Level                *string `json:"level"`
	Read                 *bool   `json:"read"`
	Saved                *bool   `json:"saved"`
	UserInterestOverride *string `json:"user_interest_override"`
}
```

Add storage methods to update saved and override fields.

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/server -run TestHandleUpdateItem_Patch -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/server/server.go internal/server/server_test.go internal/storage/storage.go
git commit -m "feat(api): support saved toggle and user interest override"
```

### Task 5: Add search endpoint with query filters

**Files:**
- Modify: `internal/server/server.go`
- Modify: `internal/server/server_test.go`
- Modify: `internal/storage/storage.go`

**Step 1: Write failing test**

```go
func TestHandleSearchItems_FilterBySavedAndLevel(t *testing.T) {
	// GET /api/items/search?q=ai&saved=true&level=high_interest
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/server -run TestHandleSearchItems_FilterBySavedAndLevel -v`  
Expected: FAIL (route missing).

**Step 3: Write minimal implementation**

Add route `/api/items/search`, parse query params, call `storage.SearchItems`.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/server -run TestHandleSearchItems_FilterBySavedAndLevel -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/server/server.go internal/server/server_test.go internal/storage/storage.go
git commit -m "feat(api): add searchable items endpoint with filters"
```

### Task 6: Add digest endpoint for weekly review

**Files:**
- Modify: `internal/server/server.go`
- Modify: `internal/server/server_test.go`
- Modify: `internal/storage/storage.go`

**Step 1: Write failing test**

```go
func TestHandleDigest_ReturnsSavedAndHighInterest(t *testing.T) {}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/server -run TestHandleDigest_ReturnsSavedAndHighInterest -v`  
Expected: FAIL.

**Step 3: Write minimal implementation**

Add `/api/digest` endpoint and storage query for:
- saved items, plus
- `high_interest` items from last 7 days.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/server -run TestHandleDigest_ReturnsSavedAndHighInterest -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/server/server.go internal/server/server_test.go internal/storage/storage.go
git commit -m "feat(api): add weekly digest endpoint"
```

### Task 7: Preserve user override precedence in engine write path

**Files:**
- Modify: `internal/engine/engine.go`
- Modify: `internal/engine/engine_test.go`
- Modify: `internal/storage/storage.go`

**Step 1: Write failing test**

```go
func TestProcessItem_DoesNotOverwriteUserInterestOverride(t *testing.T) {
	// existing item has UserInterestOverride=interest
	// new classification is uninterested
	// persisted effective level remains interest
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/engine -run TestProcessItem_DoesNotOverwriteUserInterestOverride -v`  
Expected: FAIL.

**Step 3: Write minimal implementation**

During save/update, compute effective level:
- if `UserInterestOverride != nil`, use it;
- else use AI level.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/engine -run TestProcessItem_DoesNotOverwriteUserInterestOverride -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/engine/engine.go internal/engine/engine_test.go internal/storage/storage.go
git commit -m "feat(engine): honor user interest overrides during processing"
```

### Task 8: Extend frontend types and API client for saved/search/digest

**Files:**
- Modify: `web/src/types.ts`
- Modify: `web/src/api.ts`

**Step 1: Add compile-first guard**

Run: `cd web && npm run build`  
Expected: PASS baseline before edits.

**Step 2: Write minimal implementation**

Add `Saved`, `SavedAt`, `UserInterestOverride`, `DuplicateOf` to `Item` type.  
Add API methods:
- `searchItems(params)`
- `getDigest()`
- `updateItem(id, { saved, user_interest_override, ... })`

**Step 3: Run build to verify**

Run: `cd web && npm run build`  
Expected: PASS.

**Step 4: Commit**

```bash
git add web/src/types.ts web/src/api.ts
git commit -m "feat(web): extend item typing and API client for second-brain flows"
```

### Task 9: Add save toggle and override controls in reader cards

**Files:**
- Modify: `web/src/components/ItemCard.tsx`
- Modify: `web/src/components/Reader.tsx`

**Step 1: Build before edits**

Run: `cd web && npm run build`  
Expected: PASS baseline.

**Step 2: Write minimal implementation**

In `ItemCard`:
- Add `Save/Unsave` button.
- Add override control (clearable).
- Keep optimistic UI with rollback on API error.

In `Reader`:
- Keep refresh-on-update behavior.

**Step 3: Run build to verify**

Run: `cd web && npm run build`  
Expected: PASS.

**Step 4: Commit**

```bash
git add web/src/components/ItemCard.tsx web/src/components/Reader.tsx
git commit -m "feat(web): add save and interest override controls"
```

### Task 10: Add search and filter UX in reader view

**Files:**
- Modify: `web/src/components/Reader.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/App.css`

**Step 1: Build before edits**

Run: `cd web && npm run build`  
Expected: PASS baseline.

**Step 2: Write minimal implementation**

Add:
- Search input.
- Filters: source, level, saved.
- Optional tab-like view switch: `All`, `Saved`, `Digest`.

Wire to `api.searchItems` and `api.getDigest`.

**Step 3: Run build to verify**

Run: `cd web && npm run build`  
Expected: PASS.

**Step 4: Commit**

```bash
git add web/src/components/Reader.tsx web/src/App.tsx web/src/App.css
git commit -m "feat(web): add search filters and saved/digest views"
```

### Task 11: Add migration compatibility test for existing DBs

**Files:**
- Modify: `internal/storage/storage_test.go`

**Step 1: Write failing test**

```go
func TestInitDB_UpgradesLegacyItemsTable(t *testing.T) {
	// create legacy schema without new columns
	// call InitDB
	// assert new columns exist and reads/writes still work
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/storage -run TestInitDB_UpgradesLegacyItemsTable -v`  
Expected: FAIL.

**Step 3: Write minimal implementation**

Add schema migration path in `InitDB` with `ALTER TABLE` guards for new columns and FTS setup.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/storage -run TestInitDB_UpgradesLegacyItemsTable -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/storage/storage.go internal/storage/storage_test.go
git commit -m "feat(storage): add legacy schema migration for second-brain columns"
```

### Task 12: Final verification and docs update

**Files:**
- Modify: `README.md`
- Modify: `config.json` (only if example fields are added)

**Step 1: Run full verification**

Run: `go test ./... -v`  
Expected: PASS.

Run: `cd web && npm run build`  
Expected: PASS.

**Step 2: Update docs**

Document:
- saved workflow,
- search filters,
- digest endpoint usage.

**Step 3: Final commit**

```bash
git add README.md config.json
git commit -m "docs: describe second-brain capture and retrieval workflow"
```

## Skills to apply during execution

- `@test-driven-development` for each behavior change.
- `@golang-testing` for table-driven and integration-style handler tests.
- `@frontend-patterns` for predictable React state and optimistic updates.
- `@verification-before-completion` before claiming each phase complete.

## Definition of done

- Backend tests pass: `go test ./... -v`.
- Frontend builds cleanly: `cd web && npm run build`.
- User can save, override interest, search, and view digest in Web UI.
- Existing DB users can upgrade without manual schema resets.

