# ingest-clause-linking

**Date:** 2026-04-20
**Status:** Ready to implement

## Summary

Enhances the ingestion pipeline to detect subordinate/context clauses and comma-separated object lists before the LLM call, splitting them into structured `Clause{Text, Role, SiblingIdx}` records. Context clauses are forwarded to the extractor as `contextAnnotations` (Channel 1) or appended to Description when the LLM omits them (Channel 3 fallback). Pre-filtered embedding candidates are also wired through to the extractor so the LLM can emit `EntityResolution.action="update"` against existing tasks.

## In Scope

- `Clause{Text, Role, SiblingIdx}` type and `SplitClausesWithRoles()` API in `cleanup.go`
- `expandCommaList()` helper: distributes leading verb across comma/and-separated object list
- `detectSubordinateClause()` helper: detects `[when|while|after|before|once|whenever|if] + [I|we]` openers with action verb in subordinate segment
- `Extractor.ExtractText` signature extended with `candidates []EntityMatch` and `contextAnnotations []string`
- `BuildContextAnnotationsBlock` prompt helper; list-expansion rule added to both extraction prompt templates
- Orchestrator wiring in `processTextInput`: clause-role-aware split, context annotation gathering, candidate pass-through, Channel 3 Description fallback
- All 5 implementors + 2 test mocks updated atomically with the interface change
- Unit tests (cleanup, prompt) and integration tests (orchestrator) for all new behaviors

## Out of Scope

- **Channel 2 (task-dependency edges):** filed as planner-eeeb (DependencyKind enum) — prerequisite-only model today cannot express batch/trigger relationships
- **planner-cnay:** schema dedup work
- **planner-v4hz:** prompt dedup work
- Voice-to-text comma recovery (LLM prompt rule is the fallback; no splitter fix)
- Compound noun disambiguation (no catalog; known limitation)

## Design Decisions

**1. SplitClauses return type.** `SplitClausesWithRoles()` returns `[]Clause{Text, Role, SiblingIdx}` to preserve context text for downstream channels. Role values are `main` and `context`. Context clauses carry the index of their sibling main clause via `SiblingIdx`. The old `SplitClauses() []string` signature stays for a deprecation window; `ingestbus.go:820` migrates to the new API.

**2. Context clause fate (Channels 1 + 3).** Channel 1: context clause texts are passed as `contextAnnotations []string` to `ExtractText`. The extractor prompt instructs the LLM to incorporate them into the main action item's title/description/priority/deadline but must NOT produce a separate `action_item` for the context text. Channel 3: after extraction, if the created task's `Description == ""` and a context annotation exists, the orchestrator appends the context clause text to `Description`. Channel 2 is out of scope (see above).

**3. Embedding candidate pass-through (F).** `embeddingBus.Search` is already called at `ingestbus.go:845` with results discarded. This plan wires those results (filtered at `minCandidateSimilarity = 0.70`) through to `ExtractText` as `candidates []EntityMatch`. The extractor prompt gains a `BuildCandidateBlock` section (helper already exists). The LLM emits `EntityResolution.action="update"` for similarity >= 0.85, ambiguous for 0.70–0.85; the existing processing path at `ingestbus.go:1037` handles update-vs-create with no new logic required.

**4. Subordinate clause detection guard.** Rule: clause opener matches `[when|while|after|before|once|whenever|if] + [I|we]` AND the subordinate segment itself contains an action verb (reuses `hasActionVerb`). `"if possible, call john"` falls through — no action verb in the subordinate segment — and is handled by standard splitting. This prevents false positives without a grammar parser.

## Implementation Plan

### Phase 1: Splitter enhancements (pure Go, no dependencies)

**Goal:** Introduce the `Clause` type and new splitting API; add comma-list expansion and subordinate-clause detection. Zero impact on callers until Phase 3 wires the new API.

**Files:**
- `business/domain/ingestbus/cleanup/cleanup.go`
  - Add `type Clause struct { Text string; Role string; SiblingIdx int }` and role constants `RoleMain`, `RoleContext`
  - Add `SplitClausesWithRoles(text string) []Clause` — runs list-expansion first, then subordinate detection
  - Add `expandCommaList(text string) []string` — detect leading verb (first-word check using `actionVerbs`); if comma or ", and" present in object position, distribute verb over each object; guard: no comma → return `[]string{text}` (preserves existing `"buy milk and eggs"` → 1 clause behavior)
  - Add `detectSubordinateClause(text string) (subordinate, main string, ok bool)` — match opener tokens, then call `hasActionVerb` on subordinate segment only
- `business/domain/ingestbus/cleanup/cleanup_test.go`
  - Subordinate split: `"when I go get cat litter, I also want to buy a mat"` → 2 clauses (context idx=1, main)
  - Subordinate guard: `"if possible, call john"` → 1 clause
  - Comma expansion: `"buy belt, lotion, and mat"` → 3 main clauses; `"pick up milk, eggs, and bread"` → 3 main clauses
  - Comma guard: `"buy milk and eggs"` → 1 clause (no comma; existing behavior preserved)
  - Existing split: `"buy milk and call john"` → 2 clauses (existing behavior preserved)

### Phase 2: Extractor interface + prompt update

**Goal:** Extend `Extractor.ExtractText` signature with `candidates` and `contextAnnotations`; update prompts; update all 5 implementors and 2 mocks atomically (compiler enforces completeness).

**Files:**
- `extractor/model.go` — add `candidates []EntityMatch, contextAnnotations []string` to `ExtractText` (line 8)
- `extractor/prompt.go`
  - Add `BuildContextAnnotationsBlock(annotations []string) string` — empty input returns `""`; non-empty renders as a labeled block
  - Update `buildTaskExtractionPrompt` (~line 117) and `buildGenericTextExtractionPrompt` (~line 70) to: (a) call `BuildContextAnnotationsBlock` for annotations, (b) call `BuildCandidateBlock` for candidates (already exists ~line 19), (c) add list-expansion rule to the Rules section
- `extractor/claudecli.go` — `ExtractText` (~line 402) accepts new params; passes to `BuildTextExtractionPrompt`
- `extractor/ollama.go` — match new signature; pass through (or ignore if not supported)
- `extractor/failover.go` — forward both params to primary and fallback
- `extractor/router.go` — forward both params to chosen sub-extractor
- `extractor/mock.go` — accept new params; support optional per-input-text keyed result map for multi-clause test fixtures
- `extractor/prompt_test.go` — `BuildContextAnnotationsBlock`: empty → `""`, non-empty → expected rendered block
- `extractor/claudecli_prompt_test.go` — assert task and generic prompts contain list-expansion rule text and annotations block
- `extractor/router_test.go`, `extractor/failover_test.go` — update mocks to match new signature

### Phase 3: Orchestrator wiring

**Goal:** Replace the `SplitClauses` call in `processTextInput` with `SplitClausesWithRoles`; gather context annotations per main clause; wire pre-filtered embedding candidates; implement Channel 3 Description fallback.

**Files:**
- `business/domain/ingestbus/ingestbus.go`
  - Line ~820: replace `cleanup.SplitClauses(text)` with `cleanup.SplitClausesWithRoles(text)`
  - Group context clauses by `SiblingIdx` into a `map[int][]string` before the per-clause loop
  - Add `toEntityMatches(results []embeddingbus.SearchResult) []extractor.EntityMatch` converter — maps ID, SourceType, Title, Content, Similarity; verify field names against `embeddingbus.Embedding` struct before writing
  - Per main clause: call `b.extractor.ExtractText(ctx, clause.Text, userCorrection, activeContexts, typeHint, entityMatches, contextAnnotations[clause.SiblingIdx])`
  - Channel 3: after extraction, if created task `Description == ""` and a context annotation exists, append annotation text to `Description`

### Phase 4: Integration tests

**Goal:** Validate end-to-end behavior through `processTextInput` with the real splitter + mock extractor.

**Files:**
- `business/domain/ingestbus/ingestbus_test.go`
  - `processTextSubordinateClause`: input `"when I go get cat litter, I also want to buy a mat"`, MockExtractor returns 1 `action_item` (title "Buy a mat"). Assert 1 task created.
  - `processTextCommaListExpansion`: input `"buy belt, lotion, and mat"`, MockExtractor keyed per-clause returns 1 `action_item` each. Assert 3 tasks created.
  - `processTextHighSimilarityLink`: pre-create task + seed embedding; MockExtractor returns `EntityResolution.action="update"` referencing seeded task ID. Assert 0 new tasks, existing task updated.
  - `processTextDescriptionFallback`: subordinate input, MockExtractor returns `action_item` with empty `Description`. Assert created task `Description` contains context clause text.

## Test Approach

**Splitter (Phase 1):** Table-driven tests in `cleanup_test.go`. Cover the four cases: subordinate split, subordinate guard, comma expansion, comma guard. Preserve all existing passing rows — regressions here indicate a guard misfire.

**Prompt (Phase 2):** `prompt_test.go` — `BuildContextAnnotationsBlock` unit tests (empty/non-empty). `claudecli_prompt_test.go` — string-contains assertions for list-expansion rule and annotations block presence in both prompt templates.

**Orchestrator integration (Phase 4):** Use `dbtest` + `MockExtractor` with per-input keyed results. The high-similarity-link test requires seeding an embedding via `embeddingbus` before calling `ProcessText`; the DB port is 5433 in local dev. The description-fallback test requires MockExtractor to return an empty-description `action_item` and asserts the task store reflects the annotation text appended afterward.

## Risks & Gotchas

- **`embeddingBus.Search` already exists** at `ingestbus.go:845` with results discarded — Phase 3 wires an existing call, does not add a new one. Do not duplicate the call.
- **5 implementors + 2 mocks** must all be updated in Phase 2 atomically. The compiler catches misses; do not let a PR land with a partial update.
- **`embeddingbus.SearchResult` vs `extractor.EntityMatch`** are distinct types. Verify `embeddingbus.Embedding` field names before writing the `toEntityMatches` converter in Phase 3.
- **Comma guard:** `"buy milk and eggs"` must remain 1 clause after Phase 1. The `expandCommaList` guard (no comma present) must fire before any conjunction logic.
- **Subordinate guard:** `"if possible, call john"` must remain 1 clause. `hasActionVerb` must be called on the subordinate segment only, not the full string.
- **`actionVerbs` already contains "go" and "get"** — leading-verb detection for comma-list expansion uses first-word-only check against the same map.
- **Voice-to-text drops commas** — comma-list expansion will miss those inputs. The list-expansion rule added to the LLM prompt (Phase 2) is the intended fallback; no splitter fix needed.
- **Channel 3 fires only when `Description == ""`** — do not overwrite a non-empty LLM-generated description.
- **`sqldb.ErrDBNotFound` convention** — any new store call must check and surface as `errs.NotFound`; unchecked surfaces as 500.

## Dependencies Between Phases

- Phase 1 is independent and can ship first.
- Phase 2 must come before Phase 3 (interface change required).
- Phase 3 depends on Phases 1 + 2.
- Phase 4 (integration tests) depends on all prior phases.
