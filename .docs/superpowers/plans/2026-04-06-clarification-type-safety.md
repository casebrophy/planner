# Clarification Type Safety Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate silent backend/frontend type mismatches for clarification `answerOptions` and `ClarificationKind` enum values by making Go the source of truth and generating TypeScript types from it.

**Architecture:** Typed Go structs for each clarification kind's options live in `clarificationbus/options.go`; `tygo` generates TypeScript interfaces from them. A small Go generator in `api/tooling/gen-ts-kinds/` produces the `ClarificationKindValue` TypeScript union from the live Go package. Both run under `make generate`; a pre-commit hook blocks commits when generated files are stale.

**Tech Stack:** Go, tygo (codegen), Vue 3 / TypeScript, vitest

---

## File Map

| File | Change |
|------|--------|
| `business/domain/clarificationbus/options.go` | **Create** — typed option structs + ContextRef |
| `business/domain/clarificationbus/options_test.go` | **Create** — JSON field name assertions |
| `app/domain/classifyapp/classifyapp.go` | Modify — use typed struct |
| `app/domain/mcpapp/mcpapp.go` | Modify — use typed struct |
| `business/domain/ingestbus/ingestbus.go` | Modify — use typed structs (6 sites) |
| `tools.go` | **Create** — pin tygo as Go tool dependency |
| `tygo.toml` | **Create** — codegen config |
| `api/tooling/gen-ts-kinds/main.go` | **Create** — generates ClarificationKindValue union |
| `Makefile` | Modify — add `generate` target |
| `zarf/hooks/pre-commit` | Modify — add staleness check for generated files |
| `api/services/frontend/web/src/types/generated/clarification-options.ts` | **Create (generated)** |
| `api/services/frontend/web/src/types/generated/clarification-kind.ts` | **Create (generated)** |
| `api/services/frontend/web/src/types/clarification.ts` | Modify — `ClarificationAnswerOptions` union |
| `api/services/frontend/web/src/types/enums.ts` | Modify — import generated `ClarificationKindValue` |
| `api/services/frontend/web/src/components/clarifications/ClarificationCard.vue` | Modify — use typed computeds |

---

## Task 1: Typed option structs in clarificationbus

**Files:**
- Create: `business/domain/clarificationbus/options.go`
- Create: `business/domain/clarificationbus/options_test.go`

- [ ] **Step 1: Write the failing test**

Create `business/domain/clarificationbus/options_test.go`:

```go
package clarificationbus_test

import (
	"encoding/json"
	"testing"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
)

func TestContextAssignmentOptionsJSONFields(t *testing.T) {
	opts := clarificationbus.ContextAssignmentOptions{
		SuggestedContext: "abc123",
		Confidence:       0.7,
		AvailableContexts: []clarificationbus.ContextRef{
			{ID: "ctx1", Title: "Work"},
		},
	}
	b, err := json.Marshal(opts)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"suggested_context", "confidence", "available_contexts"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing field %q", want)
		}
	}
	if _, ok := got["suggested_context_id"]; ok {
		t.Error("field suggested_context_id must not exist (old wrong name)")
	}
}

func TestNewContextOptionsJSONFields(t *testing.T) {
	opts := clarificationbus.NewContextOptions{ContextID: "id1", Title: "Work"}
	b, _ := json.Marshal(opts)
	var got map[string]any
	json.Unmarshal(b, &got)
	for _, want := range []string{"context_id", "title"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing field %q", want)
		}
	}
}

func TestAmbiguousActionOptionsJSONFields(t *testing.T) {
	opts := clarificationbus.AmbiguousActionOptions{Interpretations: []string{"a", "b"}}
	b, _ := json.Marshal(opts)
	var got map[string]any
	json.Unmarshal(b, &got)
	if _, ok := got["interpretations"]; !ok {
		t.Error("missing field interpretations")
	}
}

func TestAmbiguousDeadlineOptionsJSONFields(t *testing.T) {
	opts := clarificationbus.AmbiguousDeadlineOptions{Description: "Friday", RawDate: "friday"}
	b, _ := json.Marshal(opts)
	var got map[string]any
	json.Unmarshal(b, &got)
	for _, want := range []string{"description", "raw_date"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing field %q", want)
		}
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

```bash
go test ./business/domain/clarificationbus/... -run TestContextAssignment -count=1
```

Expected: `undefined: clarificationbus.ContextAssignmentOptions`

- [ ] **Step 3: Create options.go**

Create `business/domain/clarificationbus/options.go`:

```go
package clarificationbus

// ContextRef is a lightweight context pointer used in clarification options.
// Defined here so clarificationbus has no dependency on ingestbus/extractor.
type ContextRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ContextAssignmentOptions is the typed answer options for context_assignment clarifications.
type ContextAssignmentOptions struct {
	SuggestedContext  string       `json:"suggested_context"`
	Confidence        float64      `json:"confidence"`
	AvailableContexts []ContextRef `json:"available_contexts"`
}

// NewContextOptions is the typed answer options for new_context clarifications.
type NewContextOptions struct {
	ContextID string `json:"context_id"`
	Title     string `json:"title"`
}

// AmbiguousActionOptions is the typed answer options for ambiguous_action clarifications.
type AmbiguousActionOptions struct {
	Interpretations []string `json:"interpretations"`
}

// AmbiguousDeadlineOptions is the typed answer options for ambiguous_deadline clarifications.
type AmbiguousDeadlineOptions struct {
	Description string `json:"description"`
	RawDate     string `json:"raw_date"`
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test ./business/domain/clarificationbus/... -run "TestContextAssignment|TestNewContext|TestAmbiguousAction|TestAmbiguousDeadline" -count=1
```

Expected: `ok  github.com/casebrophy/planner/business/domain/clarificationbus`

- [ ] **Step 5: Commit**

```bash
git add business/domain/clarificationbus/options.go business/domain/clarificationbus/options_test.go
git commit -m "feat: add typed clarification option structs to clarificationbus"
```

---

## Task 2: Update clarification creation sites

**Files:**
- Modify: `app/domain/classifyapp/classifyapp.go`
- Modify: `app/domain/mcpapp/mcpapp.go`
- Modify: `business/domain/ingestbus/ingestbus.go`

The pattern for every site: replace `json.Marshal(map[string]any{...})` with `json.Marshal(clarificationbus.XxxOptions{...})`.

`ingestbus.go` also currently uses `extractor.ContextRef` when building `ctxRefs`. After this task, the same slice is used for both the extractor call (still `extractor.ContextRef`) and for building options (converted to `clarificationbus.ContextRef`). Add a helper conversion at the top of each ingest function.

- [ ] **Step 1: Update classifyapp.go (1 site)**

In `app/domain/classifyapp/classifyapp.go`, the `ctxRefs` variable is already `[]extractor.ContextRef`. Add a conversion to `[]clarificationbus.ContextRef` before the site, then replace the marshal:

```go
// Add this import at top if missing:
// "github.com/casebrophy/planner/business/domain/clarificationbus"

// Replace lines 98-104 (the map[string]any marshal):
busCtxRefs := make([]clarificationbus.ContextRef, len(ctxRefs))
for i, r := range ctxRefs {
    busCtxRefs[i] = clarificationbus.ContextRef{ID: r.ID, Title: r.Title}
}
optionsJSON, _ := json.Marshal(clarificationbus.ContextAssignmentOptions{
    SuggestedContext:  ctxID.String(),
    Confidence:        extraction.ContextConfidence,
    AvailableContexts: busCtxRefs,
})
```

- [ ] **Step 2: Run make test**

```bash
make test
```

Expected: all tests pass (no compilation errors)

- [ ] **Step 3: Update mcpapp.go (1 site)**

In `app/domain/mcpapp/mcpapp.go` at lines ~2557–2563, same pattern. The `ctxRefs` in mcpapp is also `[]extractor.ContextRef`. Add the conversion and replace the marshal:

```go
busCtxRefs := make([]clarificationbus.ContextRef, len(ctxRefs))
for i, r := range ctxRefs {
    busCtxRefs[i] = clarificationbus.ContextRef{ID: r.ID, Title: r.Title}
}
optionsJSON, _ := json.Marshal(clarificationbus.ContextAssignmentOptions{
    SuggestedContext:  ctxID.String(),
    Confidence:        extraction.ContextConfidence,
    AvailableContexts: busCtxRefs,
})
```

- [ ] **Step 4: Run make test**

```bash
make test
```

Expected: all tests pass

- [ ] **Step 5: Update ingestbus.go — email path (3 sites)**

In `business/domain/ingestbus/ingestbus.go`, the email ingestion function has 3 marshal sites. The `ctxRefs` variable is `[]extractor.ContextRef`. Add the conversion once near the top of the email function (after `ctxRefs` is built) and reuse for all 3 sites:

```go
// Add once near where ctxRefs is first used in processEmail:
busCtxRefs := make([]clarificationbus.ContextRef, len(ctxRefs))
for i, r := range ctxRefs {
    busCtxRefs[i] = clarificationbus.ContextRef{ID: r.ID, Title: r.Title}
}
```

Then replace the 3 marshals:

**Site 1 — new_context (lines ~251–255):**
```go
optionsJSON, _ := json.Marshal(clarificationbus.NewContextOptions{
    ContextID: newCtx.ID.String(),
    Title:     newCtx.Title,
})
```

**Site 2 — context_assignment (lines ~278–283):**
```go
optionsJSON, _ := json.Marshal(clarificationbus.ContextAssignmentOptions{
    SuggestedContext:  matchedContextID.String(),
    Confidence:        extraction.ContextConfidence,
    AvailableContexts: busCtxRefs,
})
```

**Site 3 — ambiguous_action (lines ~306–309):**
```go
optionsJSON, _ := json.Marshal(clarificationbus.AmbiguousActionOptions{
    Interpretations: item.Interpretations,
})
```

**Site 4 — ambiguous_deadline (lines ~336–340):**
```go
optionsJSON, _ := json.Marshal(clarificationbus.AmbiguousDeadlineOptions{
    Description: dl.Description,
    RawDate:     dl.Date,
})
```

- [ ] **Step 6: Update ingestbus.go — text/raw_input path (3 more sites)**

The `processTextInput` function has the same 3 kinds at lines ~578–582 (new_context), ~605–610 (context_assignment), ~758–761 (ambiguous_action), ~788–792 (ambiguous_deadline). Add `busCtxRefs` conversion once near the top of `processTextInput` and replace all 4 sites with the same typed structs as Step 5.

- [ ] **Step 7: Run make test**

```bash
make test
```

Expected: all tests pass, no compilation errors

- [ ] **Step 8: Commit**

```bash
git add app/domain/classifyapp/classifyapp.go app/domain/mcpapp/mcpapp.go business/domain/ingestbus/ingestbus.go
git commit -m "feat: use typed clarification option structs at all creation sites"
```

---

## Task 3: Install tygo and generate frontend option types

**Files:**
- Create: `tools.go`
- Create: `tygo.toml`
- Modify: `Makefile`
- Create (generated): `api/services/frontend/web/src/types/generated/clarification-options.ts`

- [ ] **Step 1: Create tools.go**

Create `tools.go` at repo root:

```go
//go:build tools

package tools

import (
	_ "github.com/gzuidhof/tygo"
)
```

- [ ] **Step 2: Install tygo and tidy**

```bash
go get github.com/gzuidhof/tygo
go mod tidy
```

Expected: `go.mod` and `go.sum` updated with tygo dependency.

- [ ] **Step 3: Create tygo.toml**

Create `tygo.toml` at repo root:

```toml
[[package]]
path = "github.com/casebrophy/planner/business/domain/clarificationbus"
output_path = "api/services/frontend/web/src/types/generated/clarification-options.ts"
include_files = ["options.go"]
```

- [ ] **Step 4: Create the generated output directory**

```bash
mkdir -p api/services/frontend/web/src/types/generated
```

- [ ] **Step 5: Add generate-options target to Makefile**

In `Makefile`, after the `install-hooks` section, add:

```makefile
# ==============================================================================
# Code Generation

generate: generate-options generate-kinds
	@echo "✓ TypeScript types generated"

generate-options:
	go run github.com/gzuidhof/tygo generate
```

(Leave `generate-kinds` for Task 4 — running `make generate` will fail until then; that's fine.)

- [ ] **Step 6: Run generate-options and inspect output**

```bash
make generate-options
cat api/services/frontend/web/src/types/generated/clarification-options.ts
```

Expected output (field names come from json tags):

```typescript
// Code generated by tygo. DO NOT EDIT.

export interface ContextRef {
  id: string;
  title: string;
}
export interface ContextAssignmentOptions {
  suggested_context: string;
  confidence: number;
  available_contexts: ContextRef[];
}
export interface NewContextOptions {
  context_id: string;
  title: string;
}
export interface AmbiguousActionOptions {
  interpretations: string[];
}
export interface AmbiguousDeadlineOptions {
  description: string;
  raw_date: string;
}
```

If the output differs (e.g. field names are camelCase instead of snake_case), tygo may not be respecting json tags. In that case add `type_mappings` or verify tygo version supports json tag field names. tygo >= 0.2.x respects json tags by default.

- [ ] **Step 7: Commit**

```bash
git add tools.go tygo.toml Makefile go.mod go.sum api/services/frontend/web/src/types/generated/clarification-options.ts
git commit -m "feat: add tygo codegen for clarification option types"
```

---

## Task 4: Kind enum generator

**Files:**
- Create: `api/tooling/gen-ts-kinds/main.go`
- Create (generated): `api/services/frontend/web/src/types/generated/clarification-kind.ts`
- Modify: `Makefile` (add `generate-kinds` target)

- [ ] **Step 1: Create the generator**

Create `api/tooling/gen-ts-kinds/main.go`:

```go
// gen-ts-kinds generates a TypeScript ClarificationKindValue union type from the
// Go clarificationkind package. Run via: make generate-kinds
package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/casebrophy/planner/business/types/clarificationkind"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gen-ts-kinds <output-file>")
		os.Exit(1)
	}

	// Collect all kind string values from KindWeights (exported, exhaustive).
	values := make([]string, 0, len(clarificationkind.KindWeights))
	for k := range clarificationkind.KindWeights {
		values = append(values, k.String())
	}
	sort.Strings(values)

	f, err := os.Create(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	fmt.Fprintln(f, "// Code generated by api/tooling/gen-ts-kinds. DO NOT EDIT.")
	fmt.Fprintln(f, "// Source of truth: business/types/clarificationkind/clarificationkind.go")
	fmt.Fprintln(f, "")
	fmt.Fprint(f, "export type ClarificationKindValue =")
	for _, v := range values {
		fmt.Fprintf(f, "\n  | %q", v)
	}
	fmt.Fprintln(f, "")
}
```

- [ ] **Step 2: Verify the generator compiles**

```bash
go build ./api/tooling/gen-ts-kinds/
```

Expected: no output (success)

- [ ] **Step 3: Add generate-kinds target to Makefile**

In the `generate` section added in Task 3, add the `generate-kinds` target:

```makefile
generate-kinds:
	go run ./api/tooling/gen-ts-kinds/ api/services/frontend/web/src/types/generated/clarification-kind.ts
```

- [ ] **Step 4: Run generate-kinds and inspect output**

```bash
make generate-kinds
cat api/services/frontend/web/src/types/generated/clarification-kind.ts
```

Expected output (sorted alphabetically):

```typescript
// Code generated by api/tooling/gen-ts-kinds. DO NOT EDIT.
// Source of truth: business/types/clarificationkind/clarificationkind.go

export type ClarificationKindValue =
  | "ambiguous_action"
  | "ambiguous_deadline"
  | "context_assignment"
  | "context_debrief"
  | "inactivity_prompt"
  | "new_context"
  | "overlapping_contexts"
  | "stale_task"
  | "task_debrief"
  | "voice_reference"
```

- [ ] **Step 5: Run make generate (full)**

```bash
make generate
```

Expected: both `generate-options` and `generate-kinds` succeed, `✓ TypeScript types generated` printed.

- [ ] **Step 6: Commit**

```bash
git add api/tooling/gen-ts-kinds/main.go Makefile api/services/frontend/web/src/types/generated/clarification-kind.ts
git commit -m "feat: add gen-ts-kinds generator for ClarificationKindValue union"
```

---

## Task 5: Update frontend types and ClarificationCard

**Files:**
- Modify: `api/services/frontend/web/src/types/clarification.ts`
- Modify: `api/services/frontend/web/src/types/enums.ts`
- Modify: `api/services/frontend/web/src/components/clarifications/ClarificationCard.vue`

- [ ] **Step 1: Write a failing frontend test**

Create `api/services/frontend/web/src/__tests__/types/clarification-options.test.ts`:

```typescript
import { describe, it, expect } from 'vitest'
import type { ClarificationAnswerOptions } from '@/types/clarification'
import type { ContextAssignmentOptions } from '@/types/generated/clarification-options'

describe('ClarificationAnswerOptions', () => {
  it('ContextAssignmentOptions has correct field names', () => {
    const opts: ContextAssignmentOptions = {
      suggested_context: 'uuid-here',
      confidence: 0.7,
      available_contexts: [{ id: 'ctx1', title: 'Work' }],
    }
    // If this compiles, field names are correct
    expect(opts.suggested_context).toBe('uuid-here')
    expect(opts.available_contexts[0].title).toBe('Work')
  })

  it('ClarificationAnswerOptions is a union that accepts ContextAssignmentOptions', () => {
    const opts: ClarificationAnswerOptions = {
      suggested_context: 'uuid',
      confidence: 0.5,
      available_contexts: [],
    }
    expect(opts).toBeTruthy()
  })
})
```

- [ ] **Step 2: Run test — verify it fails**

```bash
make frontend-test 2>&1 | grep -E "FAIL|Error|Cannot find"
```

Expected: `Cannot find module '@/types/generated/clarification-options'` or `ClarificationAnswerOptions` not found

- [ ] **Step 3: Update clarification.ts**

Replace the full contents of `api/services/frontend/web/src/types/clarification.ts`:

```typescript
import type { ClarificationKind, ClarificationStatus } from './enums'
import type {
  ContextAssignmentOptions,
  NewContextOptions,
  AmbiguousActionOptions,
  AmbiguousDeadlineOptions,
} from './generated/clarification-options'

export type { ContextAssignmentOptions, NewContextOptions, AmbiguousActionOptions, AmbiguousDeadlineOptions }

export type ClarificationAnswerOptions =
  | ContextAssignmentOptions
  | NewContextOptions
  | AmbiguousActionOptions
  | AmbiguousDeadlineOptions
  | null

export interface ClarificationItem {
  id: string
  kind: ClarificationKind
  status: ClarificationStatus
  subjectType: string
  subjectId: string
  question: string
  claudeGuess?: Record<string, unknown>
  reasoning?: string
  answerOptions: ClarificationAnswerOptions
  answer?: Record<string, unknown>
  priorityScore: number
  snoozedUntil?: string
  createdAt: string
  resolvedAt?: string
}

export interface ClarificationCountResponse {
  count: number
}
```

- [ ] **Step 4: Update enums.ts — use generated ClarificationKindValue**

In `api/services/frontend/web/src/types/enums.ts`, add the import and update the `ClarificationKind` type to be derived from the generated union. Replace lines 90–102:

```typescript
import type { ClarificationKindValue } from './generated/clarification-kind'

export const ClarificationKind = {
  ContextAssignment: 'context_assignment',
  StaleTask: 'stale_task',
  AmbiguousDeadline: 'ambiguous_deadline',
  NewContext: 'new_context',
  OverlappingContexts: 'overlapping_contexts',
  AmbiguousAction: 'ambiguous_action',
  VoiceReference: 'voice_reference',
  InactivityPrompt: 'inactivity_prompt',
  ContextDebrief: 'context_debrief',
  TaskDebrief: 'task_debrief',
} as const satisfies Record<string, ClarificationKindValue>

// ClarificationKind is the authoritative TypeScript type — derived from the generated union.
// If Go adds a new kind, ClarificationKindLabels and ClarificationKindColors below will
// produce a TypeScript error until they are updated.
export type ClarificationKind = ClarificationKindValue
```

The `satisfies Record<string, ClarificationKindValue>` ensures every value in the const object is a valid kind. `export type ClarificationKind = ClarificationKindValue` means that `Record<ClarificationKind, string>` in the Labels and Colors maps becomes exhaustive — adding a new Go kind will cause a TypeScript error in those maps until they're updated.

- [ ] **Step 5: Update ClarificationCard.vue — typed computeds**

In `api/services/frontend/web/src/components/clarifications/ClarificationCard.vue`, replace the current `availableContexts`, `suggestedContextId`, and `options` computed properties with typed versions. Replace lines 25–40 (the existing computeds block):

```typescript
import type { ContextAssignmentOptions, ContextRef } from '@/types/generated/clarification-options'
import type { ClarificationAnswerOptions } from '@/types/clarification'

// ... keep existing imports above ...

const options = computed((): ClarificationAnswerOptions => {
  if (!props.item.answerOptions) return null
  return typeof props.item.answerOptions === 'string'
    ? JSON.parse(props.item.answerOptions as string)
    : props.item.answerOptions as ClarificationAnswerOptions
})

const contextAssignmentOptions = computed<ContextAssignmentOptions | null>(() => {
  if (props.item.kind !== ClarificationKind.ContextAssignment) return null
  return options.value as ContextAssignmentOptions | null
})

const availableContexts = computed<ContextRef[]>(() =>
  contextAssignmentOptions.value?.available_contexts ?? []
)

const suggestedContextId = computed<string | undefined>(() =>
  contextAssignmentOptions.value?.suggested_context
)
```

Remove the now-redundant `type ContextRef = { id: string; title: string }` inline type definition (it was defined inline in the earlier bugfix — it now comes from the import).

- [ ] **Step 6: Run frontend tests**

```bash
make frontend-test
```

Expected: all 326+ tests pass.

- [ ] **Step 7: Commit**

```bash
git add \
  api/services/frontend/web/src/types/clarification.ts \
  api/services/frontend/web/src/types/enums.ts \
  api/services/frontend/web/src/components/clarifications/ClarificationCard.vue \
  api/services/frontend/web/src/__tests__/types/clarification-options.test.ts
git commit -m "feat: use generated types in frontend clarification types and card component"
```

---

## Task 6: Pre-commit staleness check

**Files:**
- Modify: `zarf/hooks/pre-commit`

- [ ] **Step 1: Add generated file staleness check to pre-commit hook**

In `zarf/hooks/pre-commit`, add the following block just before the final `exit 0` at the bottom of the file (before the stale arch report section is fine — add it after the `while IFS= read -r file; do ... done` loop):

```bash
# ── Generated TypeScript types staleness check ────────────────────────────────
# If any Go source that feeds codegen is staged, regenerate and block if dirty.
CODEGEN_SOURCES=(
    "business/domain/clarificationbus/options.go"
    "business/types/clarificationkind/clarificationkind.go"
)
NEEDS_REGEN=false
for src in "${CODEGEN_SOURCES[@]}"; do
    if git diff --cached --name-only | grep -q "^${src}$"; then
        NEEDS_REGEN=true
        break
    fi
done

if [[ "$NEEDS_REGEN" == "true" ]]; then
    make generate > /dev/null 2>&1
    if ! git diff --exit-code api/services/frontend/web/src/types/generated/ > /dev/null 2>&1; then
        echo ""
        echo "╔══════════════════════════════════════════════════════════╗"
        echo "║  ⚠  Generated TypeScript types are stale.               ║"
        echo "╚══════════════════════════════════════════════════════════╝"
        echo ""
        echo "  Run: make generate"
        echo "  Then: git add api/services/frontend/web/src/types/generated/"
        echo ""
        exit 1
    fi
fi
```

- [ ] **Step 2: Reinstall the hook**

```bash
make install-hooks
```

Expected: `✓ pre-commit hook installed`

- [ ] **Step 3: Test the hook fires correctly**

Add a whitespace change to `options.go`, stage it, and attempt to commit:

```bash
echo "" >> business/domain/clarificationbus/options.go
git add business/domain/clarificationbus/options.go
git commit -m "test hook" 2>&1 | head -10
```

Expected: the hook runs `make generate`, detects no actual diff in the generated files (since the whitespace change doesn't affect codegen), and the commit proceeds or the hook passes. This verifies the hook runs without false positives.

Revert the test change:

```bash
git checkout business/domain/clarificationbus/options.go
```

- [ ] **Step 4: Commit**

```bash
git add zarf/hooks/pre-commit
git commit -m "chore: add generated TypeScript staleness check to pre-commit hook"
```

---

## Task 7: Final verification

- [ ] **Step 1: Run full test suite**

```bash
make test
```

Expected: all Go tests pass

- [ ] **Step 2: Run frontend tests**

```bash
make frontend-test
```

Expected: all tests pass

- [ ] **Step 3: Verify generated files are committed and clean**

```bash
make generate
git diff api/services/frontend/web/src/types/generated/
```

Expected: no diff (generated files match what's in git)

- [ ] **Step 4: Update arch docs**

```
/go-arch clarification
/go-arch ingest
/vue-arch clarification
```

Stage the updated arch files before the final push commit.

- [ ] **Step 5: Session close**

```bash
git pull --rebase
bd dolt push
git push
git status  # must show "up to date with origin"
```
