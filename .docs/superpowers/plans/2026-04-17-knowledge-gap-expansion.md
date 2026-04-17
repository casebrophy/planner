# Knowledge Gap Expansion

**Date:** 2026-04-17
**Status:** planned
**Depends on:** `.docs/plans/2026-04-17-knowledge-gap-routing-fix.md` (must land first — touches `knowledgegapbus/model.go` and Detect() adapter; overlapping edits would conflict)

## Goals

1. **Backfill admin CLI** — run gap detection across existing tasks, notes, events, and contexts on demand; support dry-run.
2. **Expand extraction depth** — emit multiple gaps per entity, configurable thresholds, richer related-entity selection, new categories as a typed enum.

## Non-goals

- New HTTP routes or MCP tools — CLI-only.
- LLM-based sanitization, prompt caching, or model routing changes (routing-fix plan handles these).
- Feedback-loop scoring from clarification answers — deferred (would be a "#3 direction" follow-up).
- Migration SQL changes — gap `Category` lives in `clarification_items.answer_options` JSON, not a CHECK column.

## Design

### 1. `gapcategory` enum type

New package `business/types/gapcategory/` mirroring the `clarificationkind` pattern exactly:

- `gapcategory.go` — unexported `category` wrapper struct + exported `Category` type alias, `Parse`/`MustParse`, `AllCategories` slice, `MarshalText`/`UnmarshalText`.
- `gapcategory_test.go` — round-trip tests for all 8 values, error on unknown, `MustParse` panic.

Initial values (8):
```
missing_contact
missing_location
missing_context
missing_dependency
missing_detail
missing_deadline    (new)
missing_stakeholder (new)
missing_outcome     (new)
```

Delete the plain-string constants from `knowledgegapbus/model.go` (`CategoryMissingContact`, etc.) — reference `gapcategory.MissingContact` instead. Change `GapCandidate.Category` from `string` → `gapcategory.Category`.

### 2. Configurable `knowledgegapbus.Business`

Replace positional constructor params with a `Config` struct. Keep the existing 4 dependencies as separate args (they're interfaces, not tunables).

```go
type Config struct {
    ConfidenceThreshold float64 // default 0.6; gaps with Confidence ≤ this are skipped
    EmbeddingLimit      int     // default 10; limit passed to embeddingBus.Search
    SimilarityThreshold float64 // default 0.5; min similarity for a result to count as "related"
}

func New(log *logger.Logger, clar ClarificationCreator, emb EmbeddingSearcher, an GapAnalyzer, cfg Config) *Business
```

Zero values fall back to defaults inside `New()`. Single production call site at `api/services/planner/main.go:253` passes `Config{}` for now (same defaults); the Config struct exists so future wiring (env vars, `mux.Config`) can override without churn.

### 3. Detect() loop restructure

Current shape (one gap per entity, index [0] related entity):
```go
for _, gap := range analysis.Gaps {
    if gap.Confidence <= 0.6 { continue }
    // dedup check
    // marshal filtered[0] into KnowledgeGapOptions
    // upsert clarification
}
```

New shape (multiple gaps, per-candidate related entity):
```go
for _, gap := range analysis.Gaps {
    if gap.Confidence <= cfg.ConfidenceThreshold { continue }
    related := pickRelatedEntity(gap.RelatedIDs, filtered) // match by ID, else filtered[0]
    // marshal related into KnowledgeGapOptions with gap.Category (now typed)
    // upsert with dedup key that includes Category
}
```

**Dedup key change**: current `(Kind, SubjectType, SubjectID)` collides on the 2nd gap. Update the duplicate check and `Upsert` to include `Category` in the uniqueness key. Options:
- **Preferred:** extend `clarificationbus.QueryFilter` with an optional `Category` filter (stored inside the JSON answer_options). Upsert already does JSON comparison; verify that path handles it.
- **Fallback:** hash `(SubjectID, Category, Question)` into the clarification's stable ID. Decision deferred to implementation — worker must verify the upsert path before picking.

### 4. Prompt update

`business/domain/ingestbus/extractor/prompt.go:322-353` — update `BuildGapAnalysisPrompt`:
- Line 343 JSON schema: enumerate all 8 categories.
- Prompt text: add short definitions per category (one line each) so the model knows when to emit each.
- Keep prompt's `confidence >= 0.5` floor — two-layer filter (prompt 0.5, code 0.6) is intentional, documented in Config struct comment.

### 5. Backfill CLI command

New file `api/tooling/admin/commands/gapbackfill.go` with `func Run(ctx, cfg, log, args) error`. Wired into `api/tooling/admin/main.go` switch as `case "gap-backfill":`.

**Flags** (parsed with `flag.NewFlagSet("gap-backfill", ...)`):
- `--dry-run` — if true, analyzer still runs, but no clarification items are written.
- `--entity-type=task|note|event|context` — optional filter; default = all four.
- `--limit=N` — stop after N entities processed; 0 = unlimited.
- `--since=DURATION` — optional (e.g. `168h`); only entities updated in the window. Keeps re-runs cheap.

**Flow:**
1. Wire logger, DB, embeddingbus, clarificationbus, extractor (same router+adapter as production), and `knowledgegapbus.New(...)`.
2. For each enabled entity type, iterate via `bus.Query(ctx, filter, orderBy, page.New(n, 100))` until `len(results) < 100`.
3. Per entity: build the same `entityContent` string that the live handlers build (title + description for tasks/events; content for notes/contexts — copy the exact concatenation from `taskapp.go`, `noteapp.go`, etc. to stay consistent).
4. Call `gapBus.DetectWithOptions(ctx, entityType, id, content, DetectOptions{DryRun: bool})` — synchronous, collect `GapDetectionResult`.
5. Log summary at the end: `processed=X candidates=Y created=Z skipped=W per_type={task:..., note:..., ...}`.

**New `DetectWithOptions` method on `Business`** (additive; keep existing `Detect` as a thin wrapper that calls `DetectWithOptions` with `DetectOptions{}`):
```go
type DetectOptions struct {
    DryRun bool
}
func (b *Business) DetectWithOptions(ctx, entityType, entityID, content string, opts DetectOptions) (GapDetectionResult, error)
```

Dry-run path: run analyzer + filter + dedup check, but skip `clarificationBus.Upsert()`. Populate `CardsCreated` as "would-create" count under dry-run, distinguished by a log line, not a struct-field change.

### 6. Update existing call sites

- `app/domain/taskapp/taskapp.go`, `noteapp.go`, `eventapp.go` — no change; they call `Detect()`, which now delegates.
- `business/domain/ingestbus/ingestbus.go` — no change; same.
- `api/services/planner/main.go:253` — add `knowledgegapbus.Config{}` fourth arg.

## File touch list

| Action | File | Why |
|---|---|---|
| CREATE | `business/types/gapcategory/gapcategory.go` | New enum |
| CREATE | `business/types/gapcategory/gapcategory_test.go` | Enum tests |
| MODIFY | `business/domain/knowledgegapbus/model.go` | `Category` → typed; drop string consts; add `Config` + `DetectOptions` |
| MODIFY | `business/domain/knowledgegapbus/knowledgegapbus.go` | `New(..., Config)`, `DetectWithOptions`, multi-gap loop, per-candidate related entity, configurable threshold/limit, dedup key includes Category |
| MODIFY | `business/domain/knowledgegapbus/knowledgegapbus_test.go` | New constructor sig; table-driven tests for multi-gap, threshold boundary, dry-run |
| MODIFY | `business/domain/ingestbus/extractor/prompt.go` | Enumerate 8 categories in `BuildGapAnalysisPrompt` + per-category definitions |
| CREATE | `api/tooling/admin/commands/gapbackfill.go` | Backfill runner |
| MODIFY | `api/tooling/admin/main.go` | Add `gap-backfill` case + help text |
| MODIFY | `api/services/planner/main.go` | Pass `Config{}` to `knowledgegapbus.New()` |

## Test plan

- **gapcategory unit tests** — mirror `clarificationkind_test.go` structure; round-trip Parse for all 8 values.
- **knowledgegapbus unit tests** — update constructor calls; add:
  - Multi-gap emission: analyzer returns 3 candidates, all above threshold → 3 clarification upserts (with distinct Category in dedup key).
  - Threshold boundary: gap at exactly threshold is excluded; one tick above is included.
  - Embedding limit propagation: mock `EmbeddingSearcher` records the `limit` arg.
  - Dry-run: analyzer invoked, `clarificationBus.Upsert` not called, result counts reflect candidates.
  - Per-candidate related entity: gap with `RelatedIDs=["uuid-b"]` picks result matching uuid-b, not result[0].
  - Fallback: gap with empty/unmatched `RelatedIDs` falls back to result[0].
- **Backfill CLI** — stub integration test `gapbackfill_test.go` using `dbtest`: seed 2 tasks + 1 note, run with `--dry-run`, assert no clarification rows created but analyzer call count matches; fine to defer if time-constrained but ship the file skeleton.
- **Prompt** — no unit test (prompt is a string); verify manually that router calls succeed after the change.

## Gotchas

- **Prerequisite plan**: routing-fix plan (`2026-04-17-knowledge-gap-routing-fix.md`) edits `model.go` (`RelatedEntitySummary.Content`) and adapter sanitize step — coordinate ordering: land routing-fix first, rebase this plan on top.
- **Dedup key migration**: if any existing clarification rows have `Kind=knowledge_gap` with the current `(SubjectType, SubjectID)` uniqueness, switching to `(SubjectType, SubjectID, Category)` may produce new duplicates for pre-existing rows. Mitigate by: the backfill CLI's first run will just upsert onto the existing rows (same Category), and subsequent runs create new rows for additional Categories. No data backfill needed.
- **Prompt 0.5 vs code 0.6**: keep two-layer filter — prompt floor is a hint to the model, code threshold is the real gate. Document in Config struct comment.
- **JSON Category**: `clarification_items.answer_options` stores the Category string inside a JSON blob; existing rows use the old flat string values. The new `gapcategory.Category` must `MarshalText`/`UnmarshalText` to the same wire strings (`missing_contact`, etc.) for backward compatibility.
- **Admin CLI auth**: none — process runs locally with DB env vars, same as `migrate` and `backfill-embeddings`.
- **Pagination sentinel**: use `len(results) < pg.RowsPerPage()`, not `== 0`, to skip an extra empty-page round trip.
- **`sqldb.ErrDBNotFound`**: wrap each `bus.Query()` call in the pagination loop with an explicit check.
- **Content assembly per entity type**: copy the exact title+description concatenation from each app-layer handler; drift would cause the backfill to generate different gaps than live-path detection.

## Build sequence

1. **gapcategory enum** — create package + tests (no upstream deps).
2. **Prompt update** — enumerate 8 categories.
3. **knowledgegapbus refactor** — `Config`, `DetectWithOptions`, multi-gap loop, dedup key. Update existing tests. Pass `Config{}` at `main.go:253`.
4. **Backfill CLI** — new `gapbackfill.go` + admin/main.go switch entry.
5. **Backfill CLI integration test** — `dbtest` happy path under `--dry-run`.
6. **Manual verification** — run `make admin ARGS="gap-backfill --dry-run --limit=10"` against seeded DB; eyeball logs.
7. **Arch doc refresh** — update `.docs/arch/knowledgegap-backend.md` once behavior is stable.

Each step lands as a separate beads issue; sequential deps (2 depends on 1, 3 depends on 1+2, etc.).
