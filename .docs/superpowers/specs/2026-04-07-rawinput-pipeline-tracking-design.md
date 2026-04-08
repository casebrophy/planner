# Raw Input Pipeline Tracking

**Date:** 2026-04-07
**Status:** Approved

## Problem

The ingest pipeline silently marks raw inputs as "processed" even when extraction fails (soft-failure design). There's no visibility into which pipeline steps completed, what was extracted, or why zero tasks/events/notes were created. The user has to guess what happened.

## Solution

Add a `result JSONB` column to `raw_inputs` that captures per-step pipeline outcomes. Build a frontend detail drawer on the existing RawInputQueueView to display this data.

## Backend

### Migration 1.22

```sql
ALTER TABLE raw_inputs ADD COLUMN result JSONB;
```

### Pipeline Result Model

Defined in `business/domain/ingestbus/`:

```go
type PipelineResult struct {
    Sanitize     *StepResult `json:"sanitize,omitempty"`
    Extraction   *StepResult `json:"extraction,omitempty"`
    ContextMatch *StepResult `json:"contextMatch,omitempty"`
    Tasks        *StepResult `json:"tasks,omitempty"`
    Events       *StepResult `json:"events,omitempty"`
    Notes        *StepResult `json:"notes,omitempty"`
}

type StepResult struct {
    Status string         `json:"status"` // "completed", "failed", "skipped"
    Detail map[string]any `json:"detail,omitempty"`
}
```

### Step Detail Examples

| Step | Detail keys |
|------|-------------|
| sanitize | `pii_findings` (int) |
| extraction | `model` (string), `action_items` (int), `events` (int), `notes` (int) |
| contextMatch | `context_id` (string), `confidence` (float), `method` ("id" / "keyword" / "auto_created") |
| tasks | `count` (int), `ids` ([]string) |
| events | `count` (int), `ids` ([]string) |
| notes | `count` (int), `ids` ([]string) |

### Files Changed

1. **`business/sdk/migrate/sql/migrate.sql`** — Migration 1.22: add `result JSONB` column
2. **`business/domain/rawinputbus/model.go`** — Add `Result *json.RawMessage` to `RawInput` and `UpdateRawInput`
3. **`business/domain/rawinputbus/stores/rawinputdb/model.go`** — Add `Result *json.RawMessage` DB field, update converters
4. **`business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go`** — Include `result` in INSERT/UPDATE SQL
5. **`app/domain/rawinputapp/model.go`** — Add `Result *json.RawMessage` to app DTO
6. **`business/domain/ingestbus/ingestbus.go`** — Define `PipelineResult`/`StepResult` types. Build result as pipeline executes. Write to raw_input via `rawinputbus.Update` before `MarkProcessed`. On extraction soft-fail, record `extraction: {status: "failed", detail: {error: "..."}}` instead of silently succeeding.
7. **`business/domain/ingestbus/ingestbus.go`** — Fix double `MarkProcessing` call: `ProcessRawInputByID` already calls it, remove the duplicate call at the top of `processTextInput`.

### Bug Fix: Double MarkProcessing

`ProcessRawInputByID` (line 481) calls `MarkProcessing`, then delegates to `processTextInput` which calls `MarkProcessing` again (line 499). Remove the call from `processTextInput` since the caller already handles it.

## Frontend

### Detail Drawer

Click a row in the ingest queue table to open a detail drawer (same pattern as TaskDetailView). Shows:

- **Header**: source type, status badge, created/processed timestamps
- **Raw content**: collapsible text block
- **Pipeline steps**: vertical list with status icon (checkmark/x/dash), step name, detail summary
- **Error**: if failed, show error message
- **Actions**: reprocess button

### Files Changed

1. **`api/services/frontend/web/src/types/rawinput.ts`** — Add `result?: PipelineResult` field and `PipelineResult`/`StepResult` types
2. **`api/services/frontend/web/src/views/RawInputQueueView.vue`** — Add row click handler (router push to `:id`), add `<router-view>` slot for drawer
3. **`api/services/frontend/web/src/views/RawInputDetailView.vue`** — New detail drawer component
4. **`api/services/frontend/web/src/router/index.ts`** — Add `/ingest-queue/:id` child route
5. **`api/services/frontend/web/src/stores/rawinputStore.ts`** — Add `fetchById` action

## Out of Scope

- Processing timeline/log (timestamped entries per step) — YAGNI for now
- Separate `ingest_steps` table — overkill for single-user observability
- Changes to claudecli.go error handling — early return on error is intentional (errors are systemic)
- Adding `notes` to the text extraction JSON schema — separate concern
