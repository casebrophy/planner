# Enums Frontend

Shared TypeScript enum constants and labels used across the frontend. Sourced from `types/enums.ts` and generated files under `types/generated/`.

## Overview

`types/enums.ts` defines constants and label/color lookup records for all domain enums. Generated files under `types/generated/` are produced by `api/tooling/gen-ts-kinds` from Go source-of-truth enum types — do not edit them manually.

## Enums

### TaskStatus
Values: `open | blocked | done | dismissed`

### TaskPriority
Values: `low | medium | high | urgent`

### TaskEnergy
Values: `low | medium | high`

### ContextStatus
Values: `active | archived`

### ContextKind
Values: `work | personal | project | area | inbox`

### ClarificationKind
Generated union from `business/types/clarificationkind/clarificationkind.go`.

Current values:
- `context_assignment` — Context Assignment
- `stale_task` — Stale Task
- `ambiguous_deadline` — Ambiguous Deadline
- `new_context` — New Context
- `overlapping_contexts` — Overlapping Contexts
- `ambiguous_action` — Ambiguous Action
- `voice_reference` — Voice Reference
- `inactivity_prompt` — Inactivity
- `context_debrief` — Debrief
- `task_debrief` — Task Debrief
- `entity_link` — Entity Link

Exported records: `ClarificationKindLabels`, `ClarificationKindColors`

### ClarificationStatus
Values: `pending | snoozed | resolved | dismissed`

## Impact Callouts

### ⚠ Adding a new ClarificationKind value
When `business/types/clarificationkind/clarificationkind.go` adds a new kind:
1. Regenerate `types/generated/clarification-kind.ts` via `api/tooling/gen-ts-kinds`
2. Add the new key to `ClarificationKind` const in `types/enums.ts`
3. Add label entry to `ClarificationKindLabels`
4. Add color entry to `ClarificationKindColors`
5. Handle the new kind in `components/clarifications/ClarificationCard.vue`

### ⚠ ClarificationKindLabels / ClarificationKindColors (`types/enums.ts`)
These records are typed as `Record<ClarificationKind, string>` — if the generated union gains a new member and the records are not updated, TypeScript compilation will fail.

## File Map

- `types/enums.ts` — All enum constants and label/color lookup records
- `types/generated/clarification-kind.ts` — Auto-generated ClarificationKindValue union (from Go)
- `types/generated/observation-kind.ts` — Auto-generated ObservationKindValue union (from Go)

## Cross-Domain Dependencies

Virtually every domain component imports from `types/enums.ts`. Changes here affect:
- `components/clarifications/ClarificationCard.vue` — uses ClarificationKind, ClarificationKindLabels, ClarificationKindColors, ClarificationStatus
- `components/tasks/TaskForm.vue` — uses TaskStatus, TaskPriority, TaskEnergy
- `components/tasks/TaskFilterBar.vue` — uses TaskStatus, TaskPriority
- `components/shared/StatusBadge.vue` — uses TaskStatus, ContextStatus
- `components/shared/PriorityIndicator.vue` — uses TaskPriority
- `components/shared/EnergyIndicator.vue` — uses TaskEnergy
