# Clarification Type Safety: tygo Codegen

**Date:** 2026-04-06  
**Status:** Approved

## Problem

`answerOptions` on `ClarificationItem` is `json.RawMessage` in Go and `Record<string, unknown>` in TypeScript. The `ClarificationKind` enum is manually duplicated between `business/types/clarificationkind/` and `types/enums.ts`. Both mismatches are silent — bugs only surface at runtime when the UI renders nothing instead of crashing.

The bug that triggered this design: the frontend checked `options.suggested_context_id` but the backend sent `options.suggested_context`. Result: context assignment cards showed only Snooze/Dismiss.

## Approach

Go structs are the canonical source of truth. `tygo` generates TypeScript types from Go packages. A `make generate` target + pre-commit staleness check enforces that generated files are always committed alongside their Go source changes.

---

## Section 1: Go Structural Changes

### 1a. Refactor `business/types/clarificationkind`

Change from a struct wrapper to a plain string type so tygo can generate a TypeScript union:

```go
// Before
type Kind struct{ name string }
var ContextAssignment = Kind{"context_assignment"}

// After
type Kind string
const (
    ContextAssignment   Kind = "context_assignment"
    StaleTask           Kind = "stale_task"
    AmbiguousDeadline   Kind = "ambiguous_deadline"
    NewContext          Kind = "new_context"
    OverlappingContexts Kind = "overlapping_contexts"
    AmbiguousAction     Kind = "ambiguous_action"
    VoiceReference      Kind = "voice_reference"
    InactivityPrompt    Kind = "inactivity_prompt"
    ContextDebrief      Kind = "context_debrief"
    TaskDebrief         Kind = "task_debrief"
)

func (k Kind) String() string { return string(k) }
```

All existing call sites using `.String()` and `==` comparisons continue to work unchanged.

### 1b. New file: `business/domain/clarificationbus/options.go`

Typed structs for the 4 kinds that currently set `answerOptions`. `ContextRef` moves here (out of `ingestbus/extractor`) to avoid `clarificationbus` depending on `ingestbus`:

```go
package clarificationbus

// ContextRef is a lightweight context reference used in clarification options.
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

### 1c. Update creation sites

Three files currently build `answerOptions` via `map[string]any`:
- `app/domain/classifyapp/classifyapp.go`
- `app/domain/mcpapp/mcpapp.go`
- `business/domain/ingestbus/ingestbus.go` (3 call sites)

Each gets updated to construct the typed struct and pass it to `json.Marshal`. The bus/store layer keeps `json.RawMessage` — no DB changes.

`ingestbus` currently imports `ContextRef` from `ingestbus/extractor` for passing to the extractor. After this change, `ingestbus/extractor` keeps its own `ContextRef` (used in extraction input/output), and `clarificationbus.ContextRef` is used only when building clarification options. The two are structurally identical; conversion is a one-liner at the call site.

---

## Section 2: tygo Configuration + `make generate`

### Tool installation

tygo is added to `tools.go` (standard Go tool dependency pattern) so it's pinned in `go.sum`:

```go
//go:build tools
package tools

import _ "github.com/gzuidhof/tygo"
```

### `tygo.toml` (repo root)

```toml
[[package]]
path = "business/types/clarificationkind"
output_path = "api/services/frontend/web/src/types/generated/clarification-kind.ts"

[[package]]
path = "business/domain/clarificationbus"
output_path = "api/services/frontend/web/src/types/generated/clarification-options.ts"
# Exclude non-option types (bus methods, Storer interface, etc.)
include_files = ["options.go"]
```

### Makefile target

```makefile
generate:
	go run github.com/gzuidhof/tygo generate
```

### Pre-commit hook staleness check

Added to the hook installed by `make install-hooks`:

```bash
make generate
if ! git diff --exit-code api/services/frontend/web/src/types/generated/ > /dev/null 2>&1; then
  echo "Generated TypeScript types are stale. Run 'make generate' and stage the result."
  exit 1
fi
```

---

## Section 3: Frontend Type Updates

### 3a. Generated files (DO NOT EDIT)

`src/types/generated/clarification-kind.ts` — tygo output:
```typescript
// Code generated by tygo. DO NOT EDIT.
export type Kind = "context_assignment" | "stale_task" | "ambiguous_deadline" | ...
```

`src/types/generated/clarification-options.ts` — tygo output:
```typescript
// Code generated by tygo. DO NOT EDIT.
export interface ContextRef { id: string; title: string }
export interface ContextAssignmentOptions { suggested_context: string; confidence: number; available_contexts: ContextRef[] }
export interface NewContextOptions { context_id: string; title: string }
export interface AmbiguousActionOptions { interpretations: string[] }
export interface AmbiguousDeadlineOptions { description: string; raw_date: string }
```

### 3b. `src/types/clarification.ts` — discriminated union

Hand-written file that imports generated types and assembles the union:

```typescript
import type {
  ContextAssignmentOptions,
  NewContextOptions,
  AmbiguousActionOptions,
  AmbiguousDeadlineOptions,
} from './generated/clarification-options'

export type ClarificationAnswerOptions =
  | ContextAssignmentOptions
  | NewContextOptions
  | AmbiguousActionOptions
  | AmbiguousDeadlineOptions
  | null

export interface ClarificationItem {
  id: string
  kind: string
  question: string
  reasoning?: string
  answerOptions: ClarificationAnswerOptions
  createdAt: string
  // ... other fields
}
```

### 3c. `src/types/enums.ts`

Remove the manually maintained `ClarificationKind` const object. Import the generated `Kind` type instead. Labels and colors stay hand-written — they're display metadata, not domain types:

```typescript
import type { Kind } from './generated/clarification-kind'
export type ClarificationKind = Kind

export const ClarificationKindLabels: Record<ClarificationKind, string> = {
  context_assignment: 'Context Assignment',
  // ...
}
```

### 3d. `ClarificationCard.vue`

The `options` computed stays but is now typed. TypeScript narrows via `item.kind` checks in the template, making each per-kind block fully type-safe. The `availableContexts` and `suggestedContextId` computeds introduced in the bug fix remain, now reading from a typed `ContextAssignmentOptions` instead of casting from `unknown`.

---

## Files Changed

| File | Change |
|------|--------|
| `business/types/clarificationkind/clarificationkind.go` | Refactor Kind to string type |
| `business/domain/clarificationbus/options.go` | New — typed option structs |
| `app/domain/classifyapp/classifyapp.go` | Use typed structs |
| `app/domain/mcpapp/mcpapp.go` | Use typed structs |
| `business/domain/ingestbus/ingestbus.go` | Use typed structs (3 sites) |
| `tools.go` | Add tygo tool dependency |
| `tygo.toml` | New — codegen config |
| `Makefile` | Add `generate` target |
| `zarf/hooks/pre-commit` | Add staleness check |
| `api/services/frontend/web/src/types/generated/clarification-kind.ts` | New — generated |
| `api/services/frontend/web/src/types/generated/clarification-options.ts` | New — generated |
| `api/services/frontend/web/src/types/clarification.ts` | Import generated types, discriminated union |
| `api/services/frontend/web/src/types/enums.ts` | Import generated Kind |
| `api/services/frontend/web/src/components/clarifications/ClarificationCard.vue` | Use typed options |

## Out of Scope

- DB schema changes (none needed — `answer_options` column stays `jsonb`)
- Adding options to the 6 kinds that currently have none (`stale_task`, `inactivity_prompt`, etc.)
- OpenAPI/Swagger documentation
