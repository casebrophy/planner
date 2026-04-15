# Enums Frontend

Shared TypeScript enum constants and labels used across the frontend. Sourced from `types/enums.ts` and generated files under `types/generated/`.

## Overview

`types/enums.ts` defines constants and label/color lookup records for all domain enums. Generated files under `types/generated/` are produced by `api/tooling/gen-ts-kinds` from Go source-of-truth enum types — do not edit them manually.

## Enums

### TaskStatus
Values: `open | blocked | done | dismissed`

Label map: `TaskStatusLabels`
Color map: `StatusColors` (shared with ContextStatus)

### TaskPriority
Values: `low | medium | high | urgent`

Label map: `TaskPriorityLabels`
Color map: `PriorityColors`

### TaskEnergy
Values: `low | medium | high`

Label map: `TaskEnergyLabels`

### ContextStatus
Values: `active | paused | closed`

Label map: `ContextStatusLabels`
Color map: `StatusColors` (shared with TaskStatus)

### ContextKind
Values: `project | area | list`

Label map: `ContextKindLabels`
Color map: `ContextKindColors`

### ClarificationKind
Generated union from `business/types/clarificationkind/clarificationkind.go`.
Constrained via `as const satisfies Record<string, ClarificationKindValue>` — TypeScript errors if the Go enum gains a new member and the object is not updated.

Current values:
- `context_assignment` — Context Assignment (`#f59e0b`)
- `stale_task` — Stale Task (`#ef4444`)
- `ambiguous_deadline` — Ambiguous Deadline (`#f97316`)
- `new_context` — New Context (`#8b5cf6`)
- `overlapping_contexts` — Overlapping Contexts (`#6366f1`)
- `ambiguous_action` — Ambiguous Action (`#f59e0b`)
- `voice_reference` — Voice Reference (`#3b82f6`)
- `inactivity_prompt` — Inactivity (`#ef4444`)
- `context_debrief` — Debrief (`#10b981`)
- `task_debrief` — Task Debrief (`#06b6d4`)
- `entity_link` — Entity Link (`#a78bfa`)
- `weekly_review` — Weekly Review (`#f59e0b`)
- `type_assignment` — Type Assignment (`#14b8a6`)
- `ambiguous_entity_match` — Ambiguous Entity Match (`#f97316`)
- `event_prep` — Event Prep (`#06b6d4`)
- `knowledge_gap` — Knowledge Gap (`#14b8a6`)

Exported records: `ClarificationKindLabels`, `ClarificationKindColors`

### ClarificationStatus
Values: `pending | snoozed | resolved | dismissed`

## File Map

- `types/enums.ts` — All enum constants and label/color lookup records
- `types/generated/clarification-kind.ts` — Auto-generated `ClarificationKindValue` union (from Go `business/types/clarificationkind/clarificationkind.go`)
- `types/generated/observation-kind.ts` — Auto-generated `ObservationKindValue` union (from Go)
- `types/index.ts` — Re-exports everything from `types/enums.ts` via `export * from './enums'`
- `__tests__/types/enums.test.ts` — Exhaustive unit tests for all enum values, labels, and colors

## Impact Callouts

### ⚠ Adding a new ClarificationKind value
When `business/types/clarificationkind/clarificationkind.go` adds a new kind:
1. Regenerate `types/generated/clarification-kind.ts` via `api/tooling/gen-ts-kinds`
2. Add the new key to `ClarificationKind` const in `types/enums.ts`
3. Add label entry to `ClarificationKindLabels` — TypeScript will error if missing (exhaustive `Record<ClarificationKind, string>`)
4. Add color entry to `ClarificationKindColors` — same enforcement
5. Handle the new kind in `components/clarifications/ClarificationCard.vue` (`v-else-if` dispatch chain)

### ⚠ ClarificationKindLabels / ClarificationKindColors (`types/enums.ts`)
Typed as `Record<ClarificationKind, string>` — if the generated union gains a new member and these records are not updated, `vue-tsc` compilation fails.

### ⚠ TaskStatus (`types/enums.ts`)
Changing or removing a value affects:
- `types/task.ts` — imports `TaskStatus` as field type on `AppTask`
- `components/tasks/TaskForm.vue` — `TaskStatus` used in select options
- `components/tasks/TaskFilterBar.vue` — `TaskStatus` used in filter chips
- `components/shared/StatusBadge.vue` — `TaskStatusLabels` looked up by status value for display
- `components/shared/StatusBadge.vue` — `StatusColors` looked up for chip background color
- `__tests__/types/enums.test.ts` — hard-codes exact values and expects length 4

### ⚠ TaskPriority (`types/enums.ts`)
Changing or removing a value affects:
- `types/task.ts` — imports `TaskPriority` as field type on `AppTask`
- `components/tasks/TaskForm.vue` — `TaskPriority` used in select options
- `components/tasks/TaskFilterBar.vue` — `TaskPriority` used in filter chips
- `components/shared/PriorityIndicator.vue` — `PriorityColors` and `TaskPriorityLabels` used for icon color and tooltip
- `views/CaptureView.vue` — `TaskPriority` used in task capture form
- `__tests__/types/enums.test.ts` — hard-codes exact values and expects length 4

### ⚠ TaskEnergy (`types/enums.ts`)
Changing or removing a value affects:
- `types/task.ts` — imports `TaskEnergy` as field type on `AppTask`
- `components/tasks/TaskForm.vue` — `TaskEnergy` used in select options
- `components/shared/EnergyIndicator.vue` — `TaskEnergyLabels` used for label display
- `views/CaptureView.vue` — `TaskEnergy` used in task capture form
- `__tests__/types/enums.test.ts` — hard-codes exact values and expects length 3

### ⚠ ContextKind (`types/enums.ts`)
Changing or removing a value affects:
- `types/context.ts` — imports `ContextKind` as field type on `AppContext`
- `components/contexts/ContextForm.vue` — `ContextKind` used in kind selector (Project / Area toggle)
- `components/clarifications/ClarificationCard.vue` — `ContextKind` used when creating a new context during context-assignment clarification
- `views/ContextDetailView.vue` — `ContextKind` imported for kind badge display

### ⚠ ContextStatus (`types/enums.ts`)
Changing or removing a value affects:
- `types/context.ts` — imports `ContextStatus` as field type on `AppContext`
- `components/contexts/ContextForm.vue` — `ContextStatus` used in status selector
- `components/contexts/ContextFilterBar.vue` — `ContextStatus` used in filter chips
- `components/shared/StatusBadge.vue` — `ContextStatusLabels` and `StatusColors` used for display

## Cross-Domain Dependencies

All files below import from `types/enums.ts` (or transitively via `types/index.ts`):

| File | Imports |
|------|---------|
| `types/task.ts` | `TaskStatus`, `TaskPriority`, `TaskEnergy` |
| `types/context.ts` | `ContextStatus`, `ContextKind` |
| `types/clarification.ts` | `ClarificationKind`, `ClarificationStatus` |
| `types/index.ts` | re-exports everything |
| `components/clarifications/ClarificationCard.vue` | `ClarificationKind`, `ClarificationKindLabels`, `ClarificationKindColors`, `ContextKind` |
| `components/tasks/TaskForm.vue` | `TaskPriority`, `TaskEnergy`, `TaskStatus` |
| `components/tasks/TaskFilterBar.vue` | `TaskStatus`, `TaskPriority` |
| `components/contexts/ContextForm.vue` | `ContextStatus`, `ContextKind` |
| `components/contexts/ContextFilterBar.vue` | `ContextStatus` |
| `components/shared/StatusBadge.vue` | `StatusColors`, `TaskStatusLabels`, `ContextStatusLabels` |
| `components/shared/PriorityIndicator.vue` | `PriorityColors`, `TaskPriorityLabels`, `TaskPriority` |
| `components/shared/EnergyIndicator.vue` | `TaskEnergyLabels`, `TaskEnergy` |
| `components/dailyplan/PlanItemCard.vue` | `PriorityColors` |
| `views/CaptureView.vue` | `TaskPriority`, `TaskEnergy` |
| `views/ContextDetailView.vue` | `ContextKind` |
| `__tests__/types/enums.test.ts` | all enums, labels, and colors exhaustively |
